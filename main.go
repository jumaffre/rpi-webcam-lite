package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"crypto/tls"
	"flag"
	"log"
	"strconv"

	"github.com/blackjack/webcam"
	"golang.org/x/crypto/acme/autocert"
)

const (
	HTTPS_SERVER_PORT_DEFAULT = 4443
	HTTP_SERVED_CLIENTS = 50

	WEBCAM_DEVICE_DEFAULT= "/dev/video0"
	WEBCAM_PIXEL_FORMAT  = 0x56595559
	WEBCAM_SIZE_WIDTH    = 1024
	WEBCAM_SIZE_HEIGHT   = 768
	WEBCAM_FRAME_TIMEOUT = 5

	CERTIFICATES_FOLDER = "certs/"
)

type settings struct {
	devMode bool
	port int
	domain string
	videoDevice string
}

var (
	s settings
)

func redirectHTTP(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "https://" + r.Host + r.RequestURI, http.StatusMovedPermanently)
}

func httpIndex(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/index.html")
	})

	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "fav.ico")
	})

	mux.Handle("/_static/", http.StripPrefix("/_static/", http.FileServer(http.Dir("_static"))))
}

func httpStream(mux* http.ServeMux, li chan *bytes.Buffer) {
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Connection from", r.RemoteAddr, r.URL)

		_, err := ValidateGoogleJWT(&r.Header)
		if err != nil {
			log.Println("JWT fail")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("User verification failed: " + err.Error()))
			return
		}

		const boundary = `boundary`
		w.Header().Set("Content-Type", `multipart/x-mixed-replace; boundary=` + boundary)
		multipartWriter := multipart.NewWriter(w)
		multipartWriter.SetBoundary(boundary)
		for {
			img := <-li
			imgBytes := img.Bytes()
			iw, err := multipartWriter.CreatePart(textproto.MIMEHeader{
				"Content-type": []string{"image/jpeg"},
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
	log.Println("Starting server at ", s.domain, " on port ", s.port)
	
	// http to https redirection
	go http.ListenAndServe(":" + strconv.Itoa(s.port), http.HandlerFunc(redirectHTTP))

	var err error
	if !s.devMode {
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(s.domain),
			Cache:      autocert.DirCache(CERTIFICATES_FOLDER),
		}
		letsEncryptPort := s.port + 1
		go http.ListenAndServe(":" + strconv.Itoa(letsEncryptPort), certManager.HTTPHandler(nil))
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
			TLSConfig: 
			},
		}
		err = http.ListenAndServeTLS(CERTIFICATES_FOLDER + "certificate.pem", CERTIFICATES_FOLDER + "key.pem", nil)
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
	flag.BoolVar(&s.devMode, "dev", false, "Development mode")
	flag.IntVar(&s.port, "port", HTTPS_SERVER_PORT_DEFAULT, "Port to listen on")
	flag.StringVar(&s.domain, "domain", "", "Domain name for TLS certs")
	flag.StringVar(&s.videoDevice, "video", WEBCAM_DEVICE_DEFAULT, "Video device, e.g. /dev/video0")
	flag.Parse()
	  
	if s.devMode {
		log.Println("Warning: Server started in development Mode")
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
		log.Println("SetImageFormat return error", err)
		return
	}

	err = cam.StartStreaming()
	if err != nil {
		log.Println("err:", err)
		return
	}

	var (
		li   chan *bytes.Buffer = make(chan *bytes.Buffer)
		fi   chan []byte        = make(chan []byte)
		back chan struct{}      = make(chan struct{})
	)

	go encodeToImage(cam, back, fi, li, w, h)

	mux := http.NewServeMux()
	httpIndex(mux)
	go httpStream(mux, li)
	go startServer(mux)

	// Finally, read frames from the camera and write to fi for encoding
	for {
		err = cam.WaitForFrame(WEBCAM_FRAME_TIMEOUT)
		if err != nil {
			log.Println(err)
			return
		}

		frame, err := cam.ReadFrame()
		if err != nil {
			log.Println(err)
			return
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
