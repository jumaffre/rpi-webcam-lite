package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
	"time"

	// "github.com/stianeikeland/go-rpio"
	"log"
	"os"

	"github.com/blackjack/webcam"
)

func httpImage(li chan *bytes.Buffer) {

	fmt.Println("Starting authenticated web server...")
	// secrets := auth.HtdigestFileProvider("projecta.htdigest")
	// authenticator := auth.NewDigestAuthenticator("/", secrets)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("connect from", r.RemoteAddr, r.URL)

		//remove stale image
		// <-li

		img := <-li

		w.Header().Set("Content-Type", "image/jpeg")

		if _, err := w.Write(img.Bytes()); err != nil {
			log.Println(err)
			return
		}

	})

	log.Println("Starting to serve...")
	err2 := http.ListenAndServeTLS(":4443", "server.crt", "server.key", nil)
	if err2 != nil {
		log.Fatal("ListenAndServerTLS", err2)
	}
}

func httpVideo(li chan *bytes.Buffer) {
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		log.Println("connect from", r.RemoteAddr, r.URL)

		//remove stale image
		<-li
		const boundary = `frame`
		w.Header().Set("Content-Type", `multipart/x-mixed-replace;boundary=`+boundary)
		multipartWriter := multipart.NewWriter(w)
		multipartWriter.SetBoundary(boundary)
		for {
			img := <-li
			image := img.Bytes()
			iw, err := multipartWriter.CreatePart(textproto.MIMEHeader{
				"Content-type":   []string{"image/jpeg"},
				"Content-length": []string{strconv.Itoa(len(image))},
			})
			if err != nil {
				log.Println(err)
				return
			}
			_, err = iw.Write(image)
			if err != nil {
				log.Println(err)
				return
			}
		}
	})

	log.Println("Starting to serve...")
	err2 := http.ListenAndServeTLS(":4443", "server.crt", "server.key", nil)
	if err2 != nil {
		log.Fatal("ListenAndServerTLS", err2)
	}
}

func main() {
	camera := "/dev/video0"
	fmt.Println("Opening camera")

	cam, err := webcam.Open(camera)
	if err != nil {
		panic(err.Error())
	}
	defer cam.Close()

	format := webcam.PixelFormat(0x56595559)
	fmt.Fprintln(os.Stderr, "format", format)

	// select frame size
	frames := cam.GetSupportedFrameSizes(format)
	for _, f := range frames {
		fmt.Fprintln(os.Stderr, f.GetString())
	}

	f, w, h, err := cam.SetImageFormat(format, 1024, 768)
	if err != nil {
		log.Println("SetImageFormat return error", err)
		return
	}

	// start streaming
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

	go encodeToImage(cam, back, fi, li, w, h, f)
	// go httpImage(li)
	go httpVideo(li)

	timeout := uint32(5) //5 seconds
	// start := time.Now()
	var fr time.Duration

	for {
		err = cam.WaitForFrame(timeout)
		if err != nil {
			log.Println(err)
			return
		}

		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			log.Println(err)
			continue
		default:
			log.Println(err)
			return
		}

		frame, err := cam.ReadFrame()
		if err != nil {
			log.Println(err)
			return
		}
		if len(frame) != 0 {

			// print framerate info every 10 seconds
			fr++
			select {
			case fi <- frame:
				<-back
			default:
			}
		}
	}
}

func encodeToImage(wc *webcam.Webcam, back chan struct{}, fi chan []byte, li chan *bytes.Buffer, w, h uint32, format webcam.PixelFormat) {

	var (
		frame []byte
		img   image.Image
	)
	for {
		bframe := <-fi
		// copy frame
		if len(frame) < len(bframe) {
			frame = make([]byte, len(bframe))
		}
		copy(frame, bframe)
		back <- struct{}{}

		yuyv := image.NewYCbCr(image.Rect(0, 0, int(w), int(h)), image.YCbCrSubsampleRatio422)
		for i := range yuyv.Cb {
			ii := i * 4
			yuyv.Y[i*2] = frame[ii]
			yuyv.Y[i*2+1] = frame[ii+2]
			yuyv.Cb[i] = frame[ii+1]
			yuyv.Cr[i] = frame[ii+3]

		}
		img = yuyv

		//convert to jpeg
		buf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, img, nil); err != nil {
			log.Fatal(err)
			return
		}

		const N = 50
		// broadcast image up to N ready clients
		nn := 0
	FOR:
		for ; nn < N; nn++ {
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
