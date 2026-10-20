#pragma once
#include <atomic>
#include <cstdint>
#include <functional>
#include <thread>

// Receives the GOOSE frames the Studio GOOSE engine (studio_goose.exe) accepted, on a
// loopback UDP port (ADR-0073). Each datagram is one origin byte, 0 for a simulated
// IED in this app and 1 for a frame heard on the network, followed by the frame.
class GooseInbox {
public:
    using Handler = std::function<void(uint8_t* frame, int size, bool from_wire)>;
    ~GooseInbox();
    // Binds 127.0.0.1 on a free port, starts the receive thread and returns the port.
    int open(Handler handler);
    void close();
private:
    void run();
    std::uintptr_t socket_handle = ~std::uintptr_t(0);
    void* readable = nullptr;  // socket event (FD_READ): the thread sleeps until a datagram
    void* closing = nullptr;   // or close() wakes it; no periodic wakes
    Handler handler;
    std::thread thread;
    std::atomic<bool> running{false};
};
