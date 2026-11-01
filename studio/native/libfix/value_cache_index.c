#include "value_cache_index.h"

#include <stdint.h>
#include <stdlib.h>
#include <string.h>

/* Open addressing with linear probing; the table doubles before it is half full. */
typedef struct {
    const char* key;
    void* entry;
    uint32_t hash;
} Slot;

struct sStudioCacheIndex {
    Slot* slots;
    size_t capacity; /* always a power of two */
    size_t count;
};

static uint32_t hash_of(const char* key)
{
    uint32_t hash = 2166136261u; /* FNV-1a */
    while (*key) {
        hash ^= (unsigned char) *key++;
        hash *= 16777619u;
    }
    return hash;
}

StudioCacheIndex StudioCacheIndex_create(void)
{
    StudioCacheIndex self = (StudioCacheIndex) calloc(1, sizeof(struct sStudioCacheIndex));
    if (self) {
        self->capacity = 64;
        self->slots = (Slot*) calloc(self->capacity, sizeof(Slot));
        if (!self->slots) {
            free(self);
            return NULL;
        }
    }
    return self;
}

static Slot* find_slot(Slot* slots, size_t capacity, const char* key, uint32_t hash)
{
    size_t mask = capacity - 1;
    size_t i = hash & mask;
    while (slots[i].key && (slots[i].hash != hash || strcmp(slots[i].key, key) != 0))
        i = (i + 1) & mask;
    return &slots[i];
}

static int grow(StudioCacheIndex self)
{
    size_t capacity = self->capacity * 2;
    Slot* slots = (Slot*) calloc(capacity, sizeof(Slot));
    if (!slots)
        return 0;
    for (size_t i = 0; i < self->capacity; i++) {
        if (self->slots[i].key)
            *find_slot(slots, capacity, self->slots[i].key, self->slots[i].hash) = self->slots[i];
    }
    free(self->slots);
    self->slots = slots;
    self->capacity = capacity;
    return 1;
}

void StudioCacheIndex_add(StudioCacheIndex self, const char* key, void* entry)
{
    if (!self || !key)
        return;
    if ((self->count + 1) * 2 > self->capacity && !grow(self)) {
        /* Out of memory: drop the index; StudioCacheIndex_ready sends lookups to the vendor map. */
        free(self->slots);
        self->slots = NULL;
        self->capacity = 0;
        return;
    }
    if (!self->slots)
        return;
    uint32_t hash = hash_of(key);
    Slot* slot = find_slot(self->slots, self->capacity, key, hash);
    if (slot->key)
        return;
    slot->key = key;
    slot->entry = entry;
    slot->hash = hash;
    self->count++;
}

int StudioCacheIndex_ready(StudioCacheIndex self)
{
    return self && self->slots;
}

void* StudioCacheIndex_get(StudioCacheIndex self, const void* key)
{
    if (!self || !self->slots || !key)
        return NULL;
    const char* text = (const char*) key;
    return find_slot(self->slots, self->capacity, text, hash_of(text))->entry;
}

void StudioCacheIndex_destroy(StudioCacheIndex self)
{
    if (self) {
        free(self->slots);
        free(self);
    }
}
