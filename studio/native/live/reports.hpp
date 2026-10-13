#pragma once
#include "mms_session.hpp"
#include <memory>
#include <vector>

// Report-control-block access on an external IED. Studio only uses a block that
// the IED shows as free (not enabled, not reserved, no owner) and gives it back
// on release. These attribute writes are the only writes studio_live can make.
struct ReportUse {
    std::string rcb;
    std::string dataset;
    std::vector<std::string> members;
    std::vector<MmsVariableSpecification*> member_specs;
    std::mutex lock;
};

class ReportAccess {
public:
    explicit ReportAccess(MmsSession& session) : session(session) {}
    ~ReportAccess();
    Json status(const Json& request);
    Json reserve(const Json& request);
    Json release(const Json& request);
    void release_all();
private:
    Json status_of(const std::string& rcb);
    MmsSession& session;
    std::map<std::string, std::unique_ptr<ReportUse>> uses;
};
