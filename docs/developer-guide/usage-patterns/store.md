# NATS JetStream Key-Value & Object Store Guide

This guide details developer usage patterns for JetStream Key-Value Store and Object Store capabilities.

---

## 1. Key-Value Store

### 1.1 Basic Key-Value Operations
- **Data Persistence**: Store, retrieve, and delete key-value pairs with automatic versioning.
- **Atomic Creation**: Create keys only if they do not already exist.

```go
// Basic Key-Value Operations (Put, Get, Delete)
kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{Bucket: "app-config"})
_ = kv.Put(ctx, "feature.dark_mode", []byte("true"))

entry, err := kv.Get(ctx, "feature.dark_mode")
if err == nil {
	log.Printf("Value: %s (Revision: %d)", string(entry.Value()), entry.Revision())
}

_ = kv.Delete(ctx, "feature.dark_mode")
```

---

### 1.2 Optimistic Concurrency Control
- **Revision Matching**: Update key values conditionally based on exact revision numbers.
- **Race Condition Protection**: Prevents concurrent updates from overwriting state changes.

```go
// Conditional update based on revision (Optimistic Concurrency)
entry, _ := kv.Get(ctx, "inventory.stock")
currentRevision := entry.Revision()

// Update succeeds only if revision matches currentRevision
newRevision, err := kv.Update(ctx, "inventory.stock", []byte("150"), currentRevision)
if err != nil {
	log.Printf("Update failed due to concurrent modification: %v", err)
}
```

---

### 1.3 Reactive Key Watching
- **Real-Time Change Feeds**: Subscribe to live key mutation events across a bucket or key pattern.
- **Event-Driven Configuration**: Automatically react to configuration changes as they occur.

```go
// Watch for real-time key changes
watcher, err := kv.Watch(ctx, "config.*")
if err == nil {
	defer watcher.Stop()
	for entry := range watcher.Updates() {
		if entry != nil {
			log.Printf("Key [%s] updated to: %s", entry.Key(), string(entry.Value()))
		}
	}
}
```

---

### 1.4 Key History & Auditing
- **Audit Revision Trail**: Inspect historical revisions and deletion tombstones for a key.

```go
// Query historical revisions for a key
history, err := kv.History(ctx, "user.settings")
if err == nil {
	for _, entry := range history {
		log.Printf("Rev %d: %s (Op: %v)", entry.Revision(), string(entry.Value()), entry.Operation())
	}
}
```

---

### 1.5 Key Expiration (TTL)
- **Automatic Key Expiry**: Automatically purge keys after a configured time-to-live duration.
- **Transient State Management**: Ideal for temporary sessions, tokens, and short-lived cache entries.

```go
// Create bucket with 30-minute key expiration
kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
	Bucket: "user-sessions",
	TTL:    30 * time.Minute,
})
_ = kv.Put(ctx, "session.user_123", []byte("active"))
```

---

### 1.6 Tombstone Cleanup
- **Storage Reclamation**: Purge deleted key tombstones to reclaim underlying storage space in the bucket.

```go
// Purge deleted key tombstones
err := kv.PurgeDeletes(ctx)
```

---

## 2. Object Store

### 2.1 Streaming Upload & Download
- **Large Blob Storage**: Store and retrieve large payloads (exceeding standard message limits) like images, documents, and media assets.
- **Stream Processing**: Upload and download objects using streaming readers to minimize memory footprint.

```go
// Streaming Upload & Download
obs, err := js.CreateObjectStore(ctx, jetstream.ObjectStoreConfig{Bucket: "media-assets"})

// Upload large file via reader
fileReader := bytes.NewReader([]byte("large file content bytes..."))
meta, err := obs.Put(ctx, jetstream.ObjectMeta{Name: "reports/q3.pdf"}, fileReader)

// Download object via stream
objResult, err := obs.Get(ctx, "reports/q3.pdf")
if err == nil {
	defer objResult.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(objResult)
	log.Printf("Downloaded %d bytes", buf.Len())
}
```

---

### 2.2 Object Metadata & Lifecycle Management
- **Metadata Queries**: Retrieve file size, checksum hashes, and modification timestamps.
- **Deletion**: Permanently delete stored objects from the bucket.

```go
// Query object metadata and delete
info, err := obs.GetInfo(ctx, "reports/q3.pdf")
if err == nil {
	log.Printf("Object %s: size=%d bytes, digest=%s", info.Name, info.Size, info.Digest)
}

_ = obs.Delete(ctx, "reports/q3.pdf")
```

---

### 2.3 Object Symbolic Links
- **Zero-Byte Aliases**: Create symbolic links or aliases pointing to existing objects without duplicating stored binary data.

```go
// Create object link alias to an existing object
info, _ := obs.GetInfo(ctx, "reports/q3.pdf")
linkMeta, err := obs.AddLink(ctx, "reports/latest_q3_alias.pdf", info)
```
