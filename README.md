# Project A :movie_camera: :house_with_garden:

Secure real-time camera stream for Raspberry Pi, in the browser (written in Go).

Main features:
- Simple setup: connect a camera to your Raspberry Pi and start the server in one simple command with Docker
- Secure HTTPS with Let's Encrypt certificates and Google OAuth authentication

## Setup and Start

- Connect a camera to your Raspberry Pi, e.g. https://www.raspberrypi.org/products/camera-module-v2/
- Download and install `docker` (see https://phoenixnap.com/kb/docker-on-raspberry-pi)
- Download and install `docker-compose` (see https://dev.to/rohansawant/installing-docker-and-docker-compose-on-the-raspberry-pi-in-5-simple-steps-3mgl)
- Create Google OAuth Client ID credentials (see https://developers.google.com/identity/sign-in/web/sign-in), specifying the private (typically `https://localhost`) or public domains of your server

Then, start the server:

```
$ cd projecta/
$ export OAUTH_CLIENT_ID="<your_google_oauth_client_id>"
$ echo "<your_trusted_google_account>@gmail.com" > 
$ docker run TODO....

# Or alternatively, 
$ docker-compose up

TODO: domain and accounts file in docker!
...
```

Open your browser (don't forget to forward the server's port!)

## Settings


```bash
$ projecta --help
Usage of ./projecta:
  -accounts string
        Path to accounts file (default "accounts")
  -dev
        Development mode (expects server cert/key in certs/ folder)
  -domain string
        Domain name for TLS certs
  -insecure
        Disable OAuth auth (Warning: Use with caution!)
  -port int
        Port to listen on (default 4443)
  -video string
        Path to video device (default "/dev/video0")
```

## TODO

- [ ] OAuth ID is not hard-coded
- [ ] CI
- [ ] Motion detection
- [ ] WebRTC frames
- [ ] Take and record snapshots, per user
- [ ] IR


