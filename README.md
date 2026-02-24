# Artifacts Service | Personal Cloud Drive 2.0

A high-performance, GPU-accelerated microservice designed to handle the heavy lifting of media processing. This service consumes raw video, audio, and image files and transforms them into web-optimized artifacts (HLS, thumbnails, and metadata).

## 🚀 Key Features

* **DB-Only Job Queue:** Engineered a single-source-of-truth architecture using PostgreSQL `FOR UPDATE SKIP LOCKED`. This eliminates state drift between in-memory channels and the database.
* **Hardware Accelerated:** Multi-vendor GPU support (NVIDIA NVENC, Intel QuickSync, AMD AMF), offloading heavy transcoding from the CPU to dedicated hardware.
* **NATS-Driven Concurrency:** Uses a lightweight, event-driven **Wake-Drain-Sleep** pattern. Workers utilize 0% CPU while idle and wake instantly via NATS tokens to process the backlog.
* **Storage Decoupled:** Downloads files via context-aware HTTP streaming (presigned URLs). This removes dependency on specific storage SDKs and allows for high-concurrency downloads with a tiny RAM footprint.
* **Automatic Cleanup:** Self-cleaning temp directory logic that ensures `.tmp` files and HLS segments are purged the moment they are persisted or failed.

---

## 🏗 Architecture Overview

The service operates on a high-efficiency processing loop:

1. **Ingestion:** A NATS event hits the `jobProducer`. It registers the job in the DB and drops a wake-up token into a local semaphore channel.
2. **The Race:** A pool of $n$ persistent workers waits on the semaphore. When a token drops, exactly one worker wakes to check the DB.
3. **Atomic Lock:** The worker executes a `SKIP LOCKED` transaction to claim the oldest `pending` job, preventing any other worker from touching it.
4. **Drain Logic:** Once awake, a worker will not sleep until the database queue is completely empty, ensuring no "lost wake-up" jobs remain in the system.
5. **Persistence:** Artifact metadata, resolution, and folder sizes are calculated and saved to the `video_artifacts` table before the original job is deleted.

---

## 🛠 Tech Stack

* **Language:** Go (1.2x)
* **Messaging:** NATS
* **Database:** PostgreSQL + GORM
* **Processing:** FFmpeg (Hardware Accelerated) & FFprobe

---

## 🚦 Getting Started

### 1. Prerequisites

Ensure you have the NATS server and FFmpeg (with hardware acceleration support) installed on your system.

**Arch Linux:**

```bash
sudo pacman -S nats-server ffmpeg

```

**Ubuntu/Debian:**

```bash
sudo apt update && sudo apt install nats-server ffmpeg

```

**Fedora:**

```bash
sudo dnf install nats-server ffmpeg

```


### 2. Running the Service

```bash
go run cmd/main.go

```

### 3. Testing

Use the included tester script to blast a sample payload into the queue:

```bash
go run cmd/tester/test.go

```

---

## 📝 TODO

* [ ] Implement real-time WebSocket progress updates for the frontend.
* [ ] Add Audio processing for streamable music artifacts (MP3/AAC).
* [ ] Image thumbnail generation using FFmpeg seek-frame extraction.
* [ ] Graceful handling of transcode cancellation and signal-based cleanup.
