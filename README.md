# Raspberry Pi Webcam Server :movie_camera: :house_with_garden:

Real-time camera stream for Raspberry Pi, in the browser (written in Go).

Main features:
- Simple setup and minimal configuration: connect a camera to your Raspberry Pi and start the server in one simple command with Docker
- Secure HTTPs with Let's Encrypt certificates and Google OAuth authentication

## Setup and Start

- Connect a camera to your Raspberry Pi, e.g. https://www.raspberrypi.org/products/camera-module-v2/
- Download and install `docker` (see https://phoenixnap.com/kb/docker-on-raspberry-pi)
- Download and install `docker-compose` (see https://dev.to/rohansawant/installing-docker-and-docker-compose-on-the-raspberry-pi-in-5-simple-steps-3mgl)
- Create Google OAuth Client ID credentials (see https://developers.google.com/identity/sign-in/web/sign-in), specifying the private (typically `https://localhost`) or public domains of your server

First, setup the environment:

```bash
$ cd rpi-webcam-lite/
$ export OAUTH_CLIENT_ID="<your_google_oauth_client_id>"
$ export ACCOUNTS_FILE_PATH=</path/to/accounts/file>
$ export DOMAIN=<your_domain_name> # Not required if started if service started in dev mode (--dev)
$ echo "<your_trusted_google_account>@gmail.com" > $ACCOUNTS_FILE_PATH
```

Then, to start the server:

```bash
$ docker-compose up
```

Open your browser and enjoy! (don't forget to forward the server's port)

Alternatively, the full `docker run` commands is:

```bash
$ docker run -p 4443:4443 -p 4444:4444 -v $ACCOUNTS_FILE_PATH:/app/accounts:ro --device /dev/video0:/dev/video0 -e OAUTH_CLIENT_ID=$OAUTH_CLIENT_ID rpi-webcam --accounts /app/accounts --domain $DOMAIN
```

In development mode (i.e. directly running the server on `localhost`, without Let's Encrypt certificates), run:

```bash
$ docker run -p 4443:4443 -p 4444:4444 -v $ACCOUNTS_FILE_PATH:/app/accounts:ro --device /dev/video0:/dev/video0 -e OAUTH_CLIENT_ID=$OAUTH_CLIENT_ID rpi-webcam --accounts /app/accounts --dev
```

## Settings

```bash
$ ./rpi-webcam --help
Usage of ./rpi-webcam:
  -accounts string
        Path to accounts file (default "accounts")
  -dev
        Development mode, using self-signed certificate instead of Let\'s Encrypt (expects server cert/key in certs/ folder)
  -domain string
        Domain name of the service
  -insecure
        Disable OAuth auth (Warning: Use with caution!)
  -port int
        Port to listen on (default 4443)
  -video string
        Path to video device (default "/dev/video0")
```

## Building the Docker Image

First, clone this repository, then:

```bash
$ cd rpi-webcam/
$ docker build -t rpi-webcam .
```

## TODO

- [ ] Reduce Docker image size
- [ ] WebRTC
- [ ] Motion detection
