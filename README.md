![banner](.github/banner.png)

A blazing fast minimal ShareX backend written in golang. This backend is designed to be as minimal as possible and therefore does not provide a built-in UI. Instead, it provides a simple API that can be used to build a UI on top of it.

## Features

+ Import existing images
+ Minimal setup & config
+ Optional image compression
+ ... and more!

## Installation

1. Download the [precompiled binary](bin/echo_vault) or compile it yourself
2. Run the binary once to generate a config file
3. Edit the config file to your liking
4. Upload the [echo_vault.service](echo_vault.service) file to the same directory as the binary
5. Allow the binary to be executed using `chmod +x echo_vault`
6. Adjust the path in the service file to the path of the binary
7. Create a symlink to the service file using `ln -s /path/to/echo_vault.service /etc/systemd/system/echo_vault.service`
8. Start the service using `service echo_vault start`
9. Configure nginx (or your reverse proxy of choice) to proxy requests to the backend
10. Configure ShareX to use the backend

![sharex](.github/sharex.png)

## Configuration

When you run the program for the first time, it will generate a `config.json` file in the same directory as the executable. You can edit this file to change the configuration.

### `base_url`

The base URL of the server. This is used to generate the URL of the uploaded image. This should be the URL of the server, including the protocol (e.g. `https://example.com`).

### `port`

The port to run the server on. This should be a number between 1 and 65535. If you are running the server behind a reverse proxy, you should set this to the port that the reverse proxy is targeting.

### `upload_token`

The token that ShareX will use to authenticate with the server. This should be a random string of characters. Default is `p4$$w0rd`, but you should change this to something else. Authentication is done by sending the token as a Bearer token in the `Authorization` header.

### `max_file_size_mb`

The maximum file size in megabytes. If the file is larger than this, the server will return an error.

## API

The backend does not provide a route to view the uploaded images. For performance reasons this should be done through a reverse proxy like nginx.

```nginx
location / {
    root /path/to/your/storage;

    # Fix for old versions not using webp
    if ($request_uri ~* ^(.+)\.(jpg|png)$) {
        set $webp_uri $1.webp;

        return 301 $webp_uri;
    }

    expires 30d;
}

location ~ ^/(upload|echos) {
    proxy_pass       http://localhost:8080;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Host            $host;
}
```

### `POST /upload`

Uploads an image to the server. The image should be sent as a multipart form with the key `upload`. A successful response will look like this:

```json
{
    "extension": "png",
    "hash": "AXHN6RKPCT",
    "url": "http://localhost:8080/AXHN6RKPCT.png"
}
```

### `GET /echos`

Lists all uploaded images, ordered by timestamp desc. Returns max 15 results. Pagination is done by sending the `page` query parameter. A successful response will look like this:

```json
[
    {
        "id": 8,
        "hash": "3ZFPMNRGFJ",
        "name": "2023-11-24 00_04_28.png",
        "extension": "png",
        "upload_size": 4818389,
        "timestamp": 1701110029
    }
]
```

### `DELETE /echos/:hash`

Deletes an uploaded image. The `:hash` parameter should be the hash of the image. A successful response will look like this:

```json
{
    "success": true
}
```

## CLI

The backend also provides a few CLI commands to manage the database.

### `echo_vault scan`

Scans the storage directory for images and adds them to the database. It also converts any existing png/jpg echos to webp (compressing them). This is useful if you already have a directory full of images and want to import them into the database. Depending on how many images you have that need to be added/converted, this command may take a while to complete.

You can also use a small bash script like this one, to convert all images in the storage to webp (decent bit faster than the go implementation):

```bash
cd storage

for file in *.{jpg,jpeg,png}; do
    if [ -e "$file" ]; then
        echo $file

        cwebp -q 80 $file -o "${file%.*}.webp" -quiet -mt && rm $file
    fi
done
```

You'd then afterwards have to update the extensions in the database yourself like so:

```sql
UPDATE echos SET extension = "webp" WHERE extension IN ("png", "jpg");
```
