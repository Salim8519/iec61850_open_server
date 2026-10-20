#include "framing.hpp"
#include <algorithm>
#include <cstdio>
#include <iostream>
#include <mutex>
#include <stdexcept>
#include <vector>
#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#include <windows.h>
#endif

namespace {
std::mutex output_mutex;
FILE* frame_output = nullptr;
}

// Vendor HAL code prints diagnostics with printf. Frames therefore use a private
// duplicate of the original stdout pipe, and descriptor 1 is redirected to stderr
// so stray text becomes a native diagnostic instead of corrupting the protocol.
void protect_frame_output() {
#ifdef _WIN32
    _setmode(_fileno(stdin), _O_BINARY);
    std::fflush(stdout);
    int private_fd = _dup(_fileno(stdout));
    if (private_fd < 0) throw std::runtime_error("Unable to reserve the frame output");
    _setmode(private_fd, _O_BINARY);
    frame_output = _fdopen(private_fd, "wb");
    _dup2(_fileno(stderr), _fileno(stdout));
#else
    frame_output = fdopen(dup(fileno(stdout)), "wb");
    dup2(fileno(stderr), fileno(stdout));
#endif
    if (!frame_output) throw std::runtime_error("Unable to open the frame output");
}

#ifdef _WIN32
// Reads exactly `size` bytes of the stdin pipe; false at end of input (the parent closed it).
static bool read_input(char* target, size_t size) {
    static HANDLE input = GetStdHandle(STD_INPUT_HANDLE);
    size_t done = 0;
    while (done < size) {
        DWORD got = 0;
        DWORD wanted = static_cast<DWORD>(std::min<size_t>(size - done, 1u << 30));
        if (!ReadFile(input, target + done, wanted, &got, nullptr) || got == 0) return false;
        done += got;
    }
    return true;
}
#else
static bool read_input(char* target, size_t size) {
    return static_cast<bool>(std::cin.read(target, static_cast<std::streamsize>(size)));
}
#endif

// Same frames and errors as before; the payload buffer is reused between requests.
bool read_frame(Json& message) {
    static std::vector<char> payload;
    unsigned char header[4];
    if (!read_input(reinterpret_cast<char*>(header), 4)) return false;
    uint32_t size = header[0] | (uint32_t(header[1]) << 8) |
        (uint32_t(header[2]) << 16) | (uint32_t(header[3]) << 24);
    if (!size || size > 16 * 1024 * 1024) throw std::runtime_error("Invalid frame length");
    if (payload.size() < size) payload.resize(size);
    if (!read_input(payload.data(), size)) throw std::runtime_error("Truncated frame");
    message = Json::parse(payload.data(), payload.data() + size);
    return true;
}

void emit_frame(const Json& message) {
    std::lock_guard<std::mutex> guard(output_mutex);
    auto payload = message.dump(-1, ' ', false, Json::error_handler_t::replace);
    uint32_t size = static_cast<uint32_t>(payload.size());
    unsigned char header[] = {static_cast<unsigned char>(size),
        static_cast<unsigned char>(size >> 8), static_cast<unsigned char>(size >> 16),
        static_cast<unsigned char>(size >> 24)};
    if (!frame_output) {
        std::cout.write(reinterpret_cast<char*>(header), 4);
        std::cout.write(payload.data(), payload.size());
        std::cout.flush();
        return;
    }
    std::fwrite(header, 1, 4, frame_output);
    std::fwrite(payload.data(), 1, payload.size(), frame_output);
    std::fflush(frame_output);
}
