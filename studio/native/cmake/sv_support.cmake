# Add the unmodified upstream Sampled Values subscriber to the library, like GOOSE.
# The SV publisher is deliberately not added: only the test-only target compiles it.
foreach(sv_target iec61850 iec61850-shared)
    target_sources(${sv_target} PRIVATE "${IEC_SOURCE}/src/sampled_values/sv_subscriber.c")
endforeach()
