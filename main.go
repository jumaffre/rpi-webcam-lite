package main

import (
	"bytes"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"log"
	"strconv"

	"github.com/blackjack/webcam"
)

const (
	HTTPS_SERVER_PORT = 4443

	WEBCAM_DEVICE        = "/dev/video0"
	WEBCAM_PIXEL_FORMAT  = 0x56595559
	WEBCAM_SIZE_WIDTH    = 1024
	WEBCAM_SIZE_HEIGHT   = 768
	WEBCAM_FRAME_TIMEOUT = 5

	HTTP_SERVED_CLIENTS = 50
)

type IndexVariables struct {
	CameraBase64   string
}

func httpIndex() {
	// Index
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		vars := IndexVariables{}
		t, err := template.ParseFiles("html/index.html")
		if err != nil {
			log.Println(err)
		}
		t.Execute(w, vars)
	})

	// Favicon
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "gmail.ico")
	})

	// Static director (css)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
}

func httpImage(li chan *bytes.Buffer) {

	http.HandleFunc("/static", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Connection from", r.RemoteAddr, r.URL)

		_, err := ValidateGoogleJWT(&r.Header)
		if err != nil {
			log.Println("JWT fail")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("User verification failed: " + err.Error()))
			return
		}

		img := <-li

		w.Header().Set("Content-Type", "image/jpeg")
		if _, err := w.Write(img.Bytes()); err != nil {
			log.Println(err)
			return
		}
	})
}

func httpVideo(li chan *bytes.Buffer) {
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
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

func startServer() {
	log.Println("Starting server on port 4443")
	err := http.ListenAndServeTLS(":4443", "server.crt", "server.key", nil)
	if err != nil {
		log.Fatal("ListenAndServerTLS", err)
	}
	log.Println("Serving...")
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
	// First, open and setup camera
	log.Println("Opening camera...")
	cam, err := webcam.Open(WEBCAM_DEVICE)
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

	// Then, setup HTTP server and encoding goroutine
	var (
		li   chan *bytes.Buffer = make(chan *bytes.Buffer)
		fi   chan []byte        = make(chan []byte)
		back chan struct{}      = make(chan struct{})
	)

	go encodeToImage(cam, back, fi, li, w, h)
	go httpImage(li)
	go httpVideo(li)
	
	httpIndex()
	go startServer()

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
