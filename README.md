![banner](.github/banner.png)

A minimal golang ShareX backend.

## Features

+ Import existing images
+ Minimal setup & config
+ Optional image compression
+ ... and more!

## Installation

TBD

## Configuration

When you run the program for the first time, it will generate a `config.json` file in the same directory as the executable. You can edit this file to change the configuration.

### `base_url`

The base URL of the server. This is used to generate the URL of the uploaded image. This should be the URL of the server, including the protocol (e.g. `https://example.com`).

### `port`

The port to run the server on. This should be a number between 1 and 65535. If you are running the server behind a reverse proxy, you should set this to the port that the reverse proxy is targeting.

### `upload_token`

The token that ShareX will use to authenticate with the server. This should be a random string of characters. Default is `p4$$w0rd`, but you should change this to something else.

### `max_file_size_mb`

The maximum file size in megabytes. If the file is larger than this, the server will return an error.