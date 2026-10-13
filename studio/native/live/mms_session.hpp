#pragma once
#include "ipc/framing.hpp"
extern "C" {
#include "iec61850_client.h"
}
#include <mutex>
#include <string>

// Read-only MMS client session to one external IED. The only writes this program
// can make are report control block reservation attributes (see reports.cpp).
class MmsSession {
public:
    ~MmsSession();
    Json connect(const Json& request);
    Json discover();
    Json read(const Json& request);
    void close();
    IedConnection handle() const { return connection; }
    std::mutex& io_lock() { return io; }
private:
    IedConnection connection = nullptr;
    std::mutex io;
};
