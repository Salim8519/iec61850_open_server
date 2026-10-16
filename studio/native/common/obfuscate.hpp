// Compile-time XOR masking for license-related string literals, so they do not
// appear as plain searchable text in the executable. This is a speed bump for
// casual inspection, not cryptography; the real protection is the signature check.
#pragma once
#include <array>
#include <cstddef>
#include <string>

namespace obf {

constexpr unsigned char key_at(std::size_t index, unsigned seed) {
    return static_cast<unsigned char>((seed * 31u + index * 131u + 0x5Au) & 0xFFu);
}

template <std::size_t N, unsigned Seed>
class Masked {
public:
    constexpr explicit Masked(const char (&text)[N]) : data_{} {
        for (std::size_t i = 0; i < N; ++i) data_[i] = static_cast<char>(text[i] ^ key_at(i, Seed));
    }
    std::string reveal() const {
        std::string out(N - 1, '\0');
        for (std::size_t i = 0; i + 1 < N; ++i) out[i] = static_cast<char>(data_[i] ^ key_at(i, Seed));
        return out;
    }
    std::wstring reveal_wide() const {
        auto narrow = reveal();
        return std::wstring(narrow.begin(), narrow.end());
    }
private:
    std::array<char, N> data_;
};

// Wipe a buffer in a way the optimizer may not remove.
inline void wipe(void* data, std::size_t size) {
    volatile unsigned char* bytes = static_cast<volatile unsigned char*>(data);
    while (size--) *bytes++ = 0;
}

}  // namespace obf

#define OBF_STR(text) ([]() { \
    static constexpr obf::Masked<sizeof(text), __LINE__> masked(text); return masked.reveal(); }())
#define OBF_WSTR(text) ([]() { \
    static constexpr obf::Masked<sizeof(text), __LINE__> masked(text); return masked.reveal_wide(); }())
