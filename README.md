![banner](.github/banner.png)

A blazing fast, minimal ShareX backend written in Go. Echo-Vault features a powerful, configurable media processing pipeline for images, videos, and GIFs. It leverages `ffmpeg` and `gifsicle` to handle transcoding and optimization, keeping the core service lean while giving you full control over your media.

## Features

- Drop-in ShareX uploader with bearer-token auth
- Configurable image processing to WebP, PNG, or JPEG
- Video transcoding and optimization (MP4, WebM, etc.) powered by ffmpeg
- Advanced GIF pipeline: convert from video, resample, downscale, reduce colors, and optimize with gifsicle
- Import existing files straight into the database with the `scan` command
- Commented `config.yml` generated on first run

## Installation

1. Download the [latest release](https://github.com/coalaura/echo-vault/releases/latest) or build from source.
2. **Install dependencies.** `ffmpeg` and `ffprobe` are required for video/GIF features. `gifsicle` is required for GIF optimization.
```bash
# Debian/Ubuntu
sudo apt install ffmpeg gifsicle

# Arch Linux
sudo pacman -S ffmpeg gifsicle
```
3. Run the binary once in the target directory to create `config.yml` and the `storage/` folder.
4. Edit `config.yml` to enable features and set your domain, port, and upload token.
5. (Optional) Install the provided [echo_vault.service](echo_vault.service) unit next to the binary.
6. Make the binary executable: `chmod +x echo_vault`.
7. Update the service file paths, symlink it into `/etc/systemd/system/`, and start it (`service echo_vault start`).
8. Point nginx (or another reverse proxy) at the backend (config below).
9. Configure ShareX to send uploads to your instance using the bearer token.

![sharex](.github/sharex.png)

## Configuration

Running Echo-Vault creates a commented `config.yml`. Adjust it and restart the service.

```yaml
server:
  url: http://localhost:8080/
  port: 8080
  token: p4$$w0rd
  max_file_size: 10
  max_concurrency: 4

images:
  format: webp
  effort: 2
  quality: 90

videos:
  enabled: false
  format: mp4
  optimize: true

gifs:
  enabled: false
  optimize: true
  effort: 2
  quality: 90
  max_colors: 256
  max_framerate: 15
  max_width: 480
```

### `server` section

| Key | Type | Default | Description |
|---|---|---|---|
| `url` | string | `http://localhost:8080/` | Public base URL used when generating response links. |
| `port` | int | `8080` | Port Echo-Vault listens on. |
| `token` | string | `p4$$w0rd` | Bearer token for uploads/management. Leave empty to disable auth. |
| `max_file_size` | int | `10` | Reject uploads larger than this limit in MB. |
| `max_concurrency` | int | `4` | Maximum concurrent uploads to process. |

### `images` section

| Key | Type | Default | Description |
|---|---|---|---|
| `format` | string | `webp` | Target format for images (`webp`, `png`, `jpeg`). |
| `effort` | int (1-3) | `2` | Speed/quality trade-off (`1`=fast, `3`=slow/small). |
| `quality` | int (1-100) | `90` | Image quality; `100` is lossless for WebP. |

### `videos` section

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Allow video uploads (requires `ffmpeg`). |
| `format` | string | `mp4` | Target format (`mp4`, `webm`, `mov`, `mkv`, `gif`). |
| `optimize` | bool | `true` | Re-encode with smaller, web-friendly settings. `false` preserves quality more closely. |

### `gifs` section

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | `false` | Allow GIF uploads (requires `ffmpeg` and `ffprobe`). |
| `optimize` | bool | `true` | Run final GIFs through `gifsicle` to reduce size (requires `gifsicle`). |
| `effort` | int (1-3) | `2` | Gifsicle optimization level (`-O<effort>`). |
| `quality`| int (1-100)| `90` | `100` is lossless. Lower values enable `gifsicle --lossy` for smaller files. |
| `max_colors`| int (2-256)| `256` | Maximum colors in the GIF palette. |
| `max_framerate`| int (1-30)| `15` | Resample GIFs to this framerate if they are higher. |
| `max_width`| int (1-1024)| `480` | Downscale GIFs to this maximum width or height. |

## API

Serve static files directly through nginx, Echo-Vault also exposes `/{hash}.{ext}` for completeness, but letting nginx handle static files is faster.

```nginx
location / {
    root /path/to/your/storage;

    expires 30d;
}

location ~ ^/(upload|echos) {
    proxy_pass       http://localhost:8080;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Host            $host;
}
```

### Authentication

All API routes under `/upload` and `/echos` expect `Authorization: Bearer <token>` (from `server.token`).

### `POST /upload`

Upload a file via multipart form (`upload=<file>`). Supported types: JPEG, PNG, GIF, WebP, MP4, WebM, MOV, MKV.

```json
{
    "change": "saved 45.28% (-1.1 MB)",
    "extension": "mp4",
    "hash": "ASODE3CEHE",
    "size": "1.4 MB",
    "sniffed": "mp4",
    "timing": {
        "read": "5.2058ms",
        "store": "5.0093ms",
        "write": "1.2276779s"
    },
    "url": "http://localhost:8080/ASODE3CEHE.mp4"
}
```

### `GET /echos/{page}`

Returns up to 15 uploads per page (1-indexed).

```json
[
    {
        "id": 21,
        "hash": "ASODE3CEHE",
        "name": "my video.mp4",
        "extension": "mp4",
        "upload_size": 2483452,
        "timestamp": 1761174760
    }
]
```

### `DELETE /echos/{hash}`

Removes both the file and its database entry. Replies with `200 OK`.

## CLI

Echo-Vault doubles as a tiny maintenance tool when invoked with commands:

### `echo-vault scan`

Walks the `storage/` directory and imports missing files into the database. Progress is logged to stdout.
