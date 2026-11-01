/*
 * Hash index over the keys of the vendor MMS value cache (ADR-0133).
 *
 * The pinned libiec61850 cache keeps one entry per logical node in a linked-list map, and
 * every lookup that misses (each parent level of a data attribute name) walks the whole list.
 * IedServer_create looks up every data attribute once, so a large IED spent about a second
 * there. The index answers the same lookups in constant time: it returns exactly the entry the
 * vendor map would (the first one added under that key) and never owns keys or entries.
 */
#ifndef STUDIO_VALUE_CACHE_INDEX_H
#define STUDIO_VALUE_CACHE_INDEX_H

#ifdef __cplusplus
extern "C" {
#endif

typedef struct sStudioCacheIndex* StudioCacheIndex;

StudioCacheIndex StudioCacheIndex_create(void);

/* Remembers key -> entry unless the key is already known (the vendor map's first entry wins). */
void StudioCacheIndex_add(StudioCacheIndex self, const char* key, void* entry);

/* 1 while the index holds every key added; 0 after an allocation failure (callers then ask the
 * vendor map, so an answer is never wrong, only slower). */
int StudioCacheIndex_ready(StudioCacheIndex self);

/* The entry for key, or NULL: the same answer as Map_getEntry on the vendor string map. */
void* StudioCacheIndex_get(StudioCacheIndex self, const void* key);

void StudioCacheIndex_destroy(StudioCacheIndex self);

#ifdef __cplusplus
}
#endif

#endif
