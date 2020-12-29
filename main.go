package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"html/template"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strconv"

	"github.com/blackjack/webcam"
	"golang.org/x/crypto/acme/autocert"
)

const (
	HTTPS_SERVER_PORT_DEFAULT = 4443
	HTTP_SERVED_CLIENTS       = 50

	WEBCAM_DEVICE_DEFAULT = "/dev/video0"
	WEBCAM_PIXEL_FORMAT   = 0x56595559
	WEBCAM_SIZE_WIDTH     = 1024
	WEBCAM_SIZE_HEIGHT    = 768
	WEBCAM_FRAME_TIMEOUT  = 5

	CERTIFICATES_FOLDER    = "certs/"
	ACCOUNTS_FILE_DEFAULT  = "accounts"
	OAUTH_CLIENT_ID_ENVVAR = "OAUTH_CLIENT_ID"
)

type settings struct {
	devMode     bool
	port        int
	domain      string
	videoDevice string
	accounts    string
	insecure    bool
}

var (
	s settings
)

func redirectHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
}

func httpIndex(mux *http.ServeMux, oauthClientID *string) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		t, err := template.ParseFiles("html/index.html")
		if err != nil {
			log.Println(err)
		}
		t.Execute(w, oauthClientID)
	})

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/_static/fav.ico")
	})

	mux.Handle("/_static/", http.StripPrefix("/_static/", http.FileServer(http.Dir("_static"))))
}

func httpStream(mux *http.ServeMux, li chan *bytes.Buffer, disableAuth bool) {
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Connection from", r.RemoteAddr, r.URL)

		if !disableAuth {
			_, err := ValidateGoogleJWT(&r.Header, s.accounts)
			if err != nil {
				log.Println("JWT fail")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("User verification failed: " + err.Error()))
				return
			}
		}

		const boundary = `boundary`
		w.Header().Set("Content-Type", `multipart/x-mixed-replace; boundary=`+boundary)
		multipartWriter := multipart.NewWriter(w)
		multipartWriter.SetBoundary(boundary)
		for {
			img := <-li
			imgBytes := img.Bytes()
			iw, err := multipartWriter.CreatePart(textproto.MIMEHeader{
				"Content-type":   []string{"image/jpeg"},
				"Content-length": []string{strconv.Itoa(len(imgBytes))},
			})
			if err != nil {
				log.Println(err)
				return
			}
			_, err = iw.Write(imgBytes)
			if err != nil {
				log.Println(err)
				return
			}
		}
	})
}

func startServer(mux *http.ServeMux) {
	log.Println("Starting server on port ", s.port)

	// http to https redirection
	go http.ListenAndServe(":"+strconv.Itoa(s.port), http.HandlerFunc(redirectHTTP))

	var err error
	if !s.devMode {
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(s.domain),
			Cache:      autocert.DirCache(CERTIFICATES_FOLDER),
		}
		letsEncryptPort := s.port + 1
		go http.ListenAndServe(":"+strconv.Itoa(letsEncryptPort), certManager.HTTPHandler(nil))
		log.Println("Started Let's Encrypt on port ", letsEncryptPort)

		server := &http.Server{
			Addr:    ":" + strconv.Itoa(s.port),
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}
		err = server.ListenAndServeTLS("", "")
	} else {
		server := &http.Server{
			Addr:    ":" + strconv.Itoa(s.port),
			Handler: mux,
		}
		err = server.ListenAndServeTLS(CERTIFICATES_FOLDER+"certificate.pem", CERTIFICATES_FOLDER+"key.pem")
	}
	if err != nil {
		log.Fatal("ListenAndServerTLS", err)
	}
}

func encodeToImage(wc *webcam.Webcam, back chan struct{}, fi chan []byte, li chan *bytes.Buffer, w uint32, h uint32) {

	var frame []byte
	for {
		// Block until a frame is available from the camera
		bframe := <-fi

		if len(frame) < len(bframe) {
			frame = make([]byte, len(bframe))
		}
		copy(frame, bframe)
		back <- struct{}{}

		buf, err := formatImage(frame, w, h)
		if err != nil {
			log.Fatal(err)
			return
		}

		// Broadcast image up to HTTP_SERVED_CLIENTS ready clients
		nn := 0
	FOR:
		for ; nn < HTTP_SERVED_CLIENTS; nn++ {
			select {
			case li <- buf:
			default:
				break FOR
			}
		}
		if nn == 0 {
			li <- buf
		}
	}
}

func main() {
	googleOauthClientID := os.Getenv(OAUTH_CLIENT_ID_ENVVAR)
	if googleOauthClientID == "" {
		log.Println("OAuth client ID should be specified via " + OAUTH_CLIENT_ID_ENVVAR + " environment variable")
		os.Exit(1)
	}

	flag.BoolVar(&s.devMode, "dev", false, "Development mode, using self-signed certificate instead of Let's Encrypt (expects server cert/key in "+CERTIFICATES_FOLDER+" folder)")
	flag.IntVar(&s.port, "port", HTTPS_SERVER_PORT_DEFAULT, "Port to listen on")
	flag.StringVar(&s.domain, "domain", "", "Domain name of the service")
	flag.StringVar(&s.videoDevice, "video", WEBCAM_DEVICE_DEFAULT, "Path to video device")
	flag.StringVar(&s.accounts, "accounts", ACCOUNTS_FILE_DEFAULT, "Path to accounts file")
	flag.BoolVar(&s.insecure, "insecure", false, "Disable OAuth auth (Warning: Use with caution!)")
	flag.Parse()

	if s.accounts == "" {
		log.Fatal("Accounts file should be specified via --accounts argument")
	}

	if s.devMode {
		log.Println("Warning: Server started in development mode")
	}

	if s.insecure {
		log.Println("Warning: Server started in insecure mode: no authentication required")
	}

	log.Println("Opening camera...")
	cam, err := webcam.Open(s.videoDevice)
	if err != nil {
		panic(err.Error())
	}
	defer cam.Close()
	log.Println("Camera successfully opened")

	format := webcam.PixelFormat(WEBCAM_PIXEL_FORMAT)
	_, w, h, err := cam.SetImageFormat(format, WEBCAM_SIZE_WIDTH, WEBCAM_SIZE_HEIGHT)
	if err != nil {
		log.Fatal("SetImageFormat return error", err)
	}

	err = cam.StartStreaming()
	if err != nil {
		log.Fatal("Error starting camera stream:", err)
	}

	var (
		li   chan *bytes.Buffer = make(chan *bytes.Buffer)
		fi   chan []byte        = make(chan []byte)
		back chan struct{}      = make(chan struct{})
	)

	go encodeToImage(cam, back, fi, li, w, h)

	mux := http.NewServeMux()
	httpIndex(mux, &googleOauthClientID)
	go httpStream(mux, li, s.insecure)
	go startServer(mux)

	// Read frames from the camera and write to fi for encoding
	for {
		err = cam.WaitForFrame(WEBCAM_FRAME_TIMEOUT)
		if err != nil {
			log.Fatal(err)
		}

		frame, err := cam.ReadFrame()
		if err != nil {
			log.Fatal(err)
		}
		if len(frame) != 0 {
			select {
			case fi <- frame:
				<-back
			default:
			}
		}
	}
}
