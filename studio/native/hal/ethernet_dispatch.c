/* Studio's Ethernet HAL (ADR-0073). "udp:" interface ids go to the loopback backend
 * toward the Studio GOOSE engine; numeric ids keep using the vendor Npcap backend
 * (Sampled Values listening and test tools). */
#include <stdlib.h>
#include <string.h>
#include "ethernet_backend.h"
#include "studio_hal.h"

struct sEthernetSocket {
    PcapSocket pcap;
    StudioUdpSocket udp;
};

/* studio_ied's event loop learns when GOOSE frames leave, to schedule retransmissions. */
static StudioFrameObserver sendObserver = NULL;
static void* sendObserverContext = NULL;

void StudioEthernet_setSendObserver(StudioFrameObserver observer, void* context)
{
    sendObserverContext = context;
    sendObserver = observer;
}

void Ethernet_getInterfaceMACAddress(const char* interfaceId, uint8_t* addr)
{
    if (StudioUdp_isInterface(interfaceId)) {
        if (!StudioUdp_getMAC(interfaceId, addr)) memset(addr, 0, 6);
        return;
    }
    Pcap_Ethernet_getInterfaceMACAddress(interfaceId, addr);
}

EthernetSocket Ethernet_createSocket(const char* interfaceId, uint8_t* destAddress)
{
    EthernetSocket self = (EthernetSocket) calloc(1, sizeof(struct sEthernetSocket));
    if (self == NULL) return NULL;
    if (StudioUdp_isInterface(interfaceId)) {
        /* Send-only: a receiver (destAddress == NULL) would busy-spin libiec61850's
         * receive loop, because the vendor handle set never waits. studio_live
         * receives the engine's frames through its own inbox instead. */
        if (destAddress != NULL) self->udp = StudioUdp_create(interfaceId);
    } else {
        self->pcap = Pcap_Ethernet_createSocket(interfaceId, destAddress);
    }
    if (self->udp == NULL && self->pcap == NULL) {
        free(self);
        return NULL;
    }
    return self;
}

void Ethernet_destroySocket(EthernetSocket self)
{
    if (self->udp) StudioUdp_destroy(self->udp);
    if (self->pcap) Pcap_Ethernet_destroySocket(self->pcap);
    free(self);
}

void Ethernet_sendPacket(EthernetSocket self, uint8_t* buffer, int packetSize)
{
    if (self->udp) StudioUdp_send(self->udp, buffer, packetSize);
    else Pcap_Ethernet_sendPacket(self->pcap, buffer, packetSize);
    if (sendObserver) sendObserver(sendObserverContext);
}

void Ethernet_setMode(EthernetSocket self, EthernetSocketMode mode)
{
    if (self->pcap) Pcap_Ethernet_setMode(self->pcap, mode);
}

void Ethernet_addMulticastAddress(EthernetSocket self, const uint8_t* multicastAddress)
{
    if (self->pcap) Pcap_Ethernet_addMulticastAddress(self->pcap, multicastAddress);
}

void Ethernet_setProtocolFilter(EthernetSocket self, uint16_t etherType)
{
    if (self->pcap) Pcap_Ethernet_setProtocolFilter(self->pcap, etherType);
}

int Ethernet_receivePacket(EthernetSocket self, uint8_t* buffer, int bufferSize)
{
    return self->pcap ? Pcap_Ethernet_receivePacket(self->pcap, buffer, bufferSize) : 0;
}

bool Ethernet_isSupported(void)
{
    return true;
}
