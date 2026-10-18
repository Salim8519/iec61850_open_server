#pragma once
// Who uses this simulated IED over MMS: one summary per second, never one event per read.
#include <atomic>
#include <condition_variable>
#include <cstdint>
#include <map>
#include <mutex>
#include <set>
#include <string>
#include <thread>

class AccessLog {
public:
    ~AccessLog();
    void start();
    void stop();
    void connection(const std::string& peer, bool connected);
    void read(const char* domain, const char* item, const char* peer);
    void write(const std::string& peer, const std::string& object);
    void report(const std::string& peer, const std::string& rcb, int event);

private:
    struct Client {
        bool connected = true;
        uint64_t total_reads = 0;
        std::map<std::string, uint64_t> reads;   // since the last summary, per LD/LN
        std::map<std::string, uint64_t> writes;  // since the last summary, per object
        std::set<std::string> rcbs;              // report blocks this client has enabled
    };
    static constexpr size_t MAX_CLIENTS = 32;
    static constexpr size_t MAX_NODES = 64;
    Client* client(const std::string& peer);
    void flush();
    std::mutex lock;
    std::condition_variable wake;
    std::map<std::string, Client> clients;
    bool dirty = false;
    std::atomic<bool> running{false};
    std::thread flusher;
};
