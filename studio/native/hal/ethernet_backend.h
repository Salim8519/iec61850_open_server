/* Backends behind the Studio Ethernet HAL dispatcher (ethernet_dispatch.c).
 * Pcap_*: the vendor Npcap HAL compiled under new names (ethernet_pcap.c).
 * StudioUdp_*: send-only loopback backend toward the Studio GOOSE engine (ADR-0073). */
#ifndef STUDIO_ETHERNET_BACKEND_H
#define STUDIO_ETHERNET_BACKEND_H

#include <stdbool.h>
#include <stdint.h>
#include "hal_ethernet.h"

typedef struct sPcapEthernetSocket* PcapSocket;
typedef struct sStudioUdpSocket* StudioUdpSocket;

void Pcap_Ethernet_getInterfaceMACAddress(const char* interfaceId, uint8_t* addr);
PcapSocket Pcap_Ethernet_createSocket(const char* interfaceId, uint8_t* destAddress);
void Pcap_Ethernet_destroySocket(PcapSocket socket);
void Pcap_Ethernet_sendPacket(PcapSocket socket, uint8_t* buffer, int packetSize);
void Pcap_Ethernet_setMode(PcapSocket socket, EthernetSocketMode mode);
void Pcap_Ethernet_addMulticastAddress(PcapSocket socket, const uint8_t* multicastAddress);
void Pcap_Ethernet_setProtocolFilter(PcapSocket socket, uint16_t etherType);
int Pcap_Ethernet_receivePacket(PcapSocket socket, uint8_t* buffer, int bufferSize);
bool Pcap_Ethernet_isSupported(void);

/* Interface ids of the form "udp:<engine port>:<source MAC as 12 hex digits>". */
bool StudioUdp_isInterface(const char* interfaceId);
bool StudioUdp_getMAC(const char* interfaceId, uint8_t* addr);
StudioUdpSocket StudioUdp_create(const char* interfaceId);
void StudioUdp_send(StudioUdpSocket socket, const uint8_t* buffer, int size);
void StudioUdp_destroy(StudioUdpSocket socket);

#endif
