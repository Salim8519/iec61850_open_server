// Modbus TCP server: MBAP framing, unit-id check and one thread per master connection.
#include "modbus_pdu.hpp"
#include "protocol.hpp"
#include "traffic.hpp"
#include <atomic>
#include <set>
#include <stdexcept>
#include <thread>
#ifdef _WIN32
#include <winsock2.h>
#include <ws2tcpip.h>
using socket_t = SOCKET;
#else
#include <arpa/inet.h>
#include <netinet/in.h>
#include <sys/socket.h>
#include <unistd.h>
using socket_t = int;
#define INVALID_SOCKET (-1)
#define closesocket close
#endif

namespace {
bool receive(socket_t socket, uint8_t* buffer, int size) {
    int done = 0;
    while (done < size) {
        int count = recv(socket, reinterpret_cast<char*>(buffer + done), size - done, 0);
        if (count <= 0) return false;
        done += count;
    }
    return true;
}

class ModbusServer : public ProtocolServer {
public:
    ModbusServer(PointTable& table, CommandBridge& bridge) : application(table, bridge) {}
    ~ModbusServer() override { stop(); }

    Json start(const Json& settings) override {
#ifdef _WIN32
        WSADATA data;
        WSAStartup(MAKEWORD(2, 2), &data);
#endif
        auto bind_ip = settings.value("bind", std::string("127.0.0.1"));
        int port = settings.value("port", 502);
        unit_id = settings.value("unit_id", 1);
        if (unit_id < 0 || unit_id > 255) throw std::runtime_error("Unit ID must be 0 to 255");
        max_connections = settings.value("max_connections", 4);
        allowed = settings.value("allowed", std::string());  // Allowed SCADA address (ADR-0124); empty: anyone
        listener = socket(AF_INET, SOCK_STREAM, IPPROTO_TCP);
        sockaddr_in address{};
        address.sin_family = AF_INET;
        address.sin_port = htons(uint16_t(port));
        if (inet_pton(AF_INET, bind_ip.c_str(), &address.sin_addr) != 1)
            throw std::runtime_error("Listen address " + bind_ip + " is not an IPv4 address");
        if (listener == INVALID_SOCKET || ::bind(listener, reinterpret_cast<sockaddr*>(&address), sizeof(address)) != 0
            || listen(listener, 8) != 0) {
            if (listener != INVALID_SOCKET) closesocket(listener);
            listener = INVALID_SOCKET;
            throw std::runtime_error("Could not listen on " + bind_ip + ":" + std::to_string(port)
                                     + ". Another program may use this port.");
        }
        running = true;
        acceptor = std::thread([this] { accept_loop(); });
        return {{"protocol", "modbus"}, {"bind", bind_ip}, {"port", port}, {"unit_id", unit_id}};
    }

    void changed(const std::vector<GatewayPoint>&) override {}  // Modbus masters poll.

    void stop() override {
        if (!running.exchange(false)) return;
        closesocket(listener);
        {
            std::lock_guard<std::mutex> guard(lock);
            for (auto client : clients) shutdown(client, 2);  // serve() closes its own socket
        }
        if (acceptor.joinable()) acceptor.join();
        for (auto& worker : workers) if (worker.joinable()) worker.join();
        workers.clear();
    }

private:
    ModbusApplication application;
    socket_t listener = INVALID_SOCKET;
    std::atomic<bool> running{false};
    std::thread acceptor;
    std::vector<std::thread> workers;
    std::mutex lock;
    std::set<socket_t> clients;
    int unit_id = 1;
    int max_connections = 4;
    std::string allowed;

    void accept_loop() {
        while (running) {
            sockaddr_in remote{};
            socklen_t size = sizeof(remote);
            auto client = accept(listener, reinterpret_cast<sockaddr*>(&remote), &size);
            if (client == INVALID_SOCKET) continue;
            char text[INET_ADDRSTRLEN] = {0};
            inet_ntop(AF_INET, &remote.sin_addr, text, sizeof(text));
            std::string peer = std::string(text) + ":" + std::to_string(ntohs(remote.sin_port));
            if (!allowed.empty() && allowed != text) {
                traffic("rx", peer, "connection refused: only the allowed SCADA address " + allowed + " may connect");
                closesocket(client);
                continue;
            }
            std::lock_guard<std::mutex> guard(lock);
            if (int(clients.size()) >= max_connections || !running) {
                traffic("rx", peer, "connection refused: the connection limit is reached");
                closesocket(client);
                continue;
            }
            clients.insert(client);
            workers.emplace_back([this, client, peer] { serve(client, peer); });
        }
    }

    void serve(socket_t client, const std::string& peer) {
        connection_event("open", peer);
        traffic("rx", peer, "Master connected");
        uint8_t header[7];
        while (running && receive(client, header, 7)) {
            int length = (header[4] << 8) | header[5];
            if (((header[2] << 8) | header[3]) != 0 || length < 2 || length > 254) {
                traffic("rx", peer, "invalid MBAP header: connection closed");
                break;
            }
            std::vector<uint8_t> request(length - 1);
            if (!receive(client, request.data(), length - 1)) break;
            std::string summary;
            std::vector<uint8_t> response;
            if (unit_id != 0 && header[6] != unit_id && header[6] != 0) {
                summary = "request for unit " + std::to_string(header[6]) + ", this gateway is unit "
                          + std::to_string(unit_id) + " (exception 11)";
                response = modbus_exception(request[0], 0x0B);
            } else {
                response = application.handle(request, peer, summary);
            }
            traffic("rx", peer, summary);
            std::vector<uint8_t> frame = {header[0], header[1], 0, 0, uint8_t((response.size() + 1) >> 8),
                                          uint8_t((response.size() + 1) & 0xFF), header[6]};
            frame.insert(frame.end(), response.begin(), response.end());
            if (send(client, reinterpret_cast<const char*>(frame.data()), int(frame.size()), 0) <= 0) break;
        }
        {
            std::lock_guard<std::mutex> guard(lock);
            if (clients.erase(client)) closesocket(client);
        }
        connection_event("closed", peer);
        traffic("rx", peer, "Master disconnected");
    }
};
}

std::unique_ptr<ProtocolServer> make_modbus(PointTable& table, CommandBridge& bridge) {
    return std::make_unique<ModbusServer>(table, bridge);
}
