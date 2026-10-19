/* The vendor's Windows Npcap Ethernet HAL, compiled unmodified under Pcap_* names
 * (ADR-0006 pattern: the vendor tree is never edited). ethernet_dispatch.c provides
 * the public Ethernet_* functions and routes numeric interface ids here. The
 * EthernetHandleSet_* functions keep their vendor names; with the vendor's
 * ETHERNET_WIN32_DISABLE_ETHERNET_HANDLESET they never touch a socket. */
#define sEthernetSocket sPcapEthernetSocket
#define Ethernet_getInterfaceMACAddress Pcap_Ethernet_getInterfaceMACAddress
#define Ethernet_createSocket Pcap_Ethernet_createSocket
#define Ethernet_destroySocket Pcap_Ethernet_destroySocket
#define Ethernet_sendPacket Pcap_Ethernet_sendPacket
#define Ethernet_setMode Pcap_Ethernet_setMode
#define Ethernet_addMulticastAddress Pcap_Ethernet_addMulticastAddress
#define Ethernet_setProtocolFilter Pcap_Ethernet_setProtocolFilter
#define Ethernet_receivePacket Pcap_Ethernet_receivePacket
#define Ethernet_isSupported Pcap_Ethernet_isSupported

#include "ethernet_win32.c"
