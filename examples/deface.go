package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/gpavlidi/between"

	"github.com/lazywei/go-opencv/opencv"
)

const (
	netInterface = "tap0"
	httpPort     = "8000"
	httpsPort    = "8001"
	certFile     = "./sslKeys/server.crt"
	keyFile      = "./sslKeys/server.key"
)

func OrDie(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func main() {
	OrDie(between.CheckCompatibility())

	log.Println("Activating Transparent Proxy...")
	OrDie(between.Configure(httpPort, httpsPort, netInterface))
	OrDie(between.Enable())

	// catch Ctrl+C and cleanup pf
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGABRT)
	go func() {
		for range c {
			log.Println("Deactivating Proxy. Network connectivity should be restored.")
			OrDie(between.Disable())
			os.Exit(3)
		}
	}()

	px := between.NewProxy()
	px.ProcessRequest = processRequest
	px.ProcessResponse = processResponse

	go http.ListenAndServe(":"+httpPort, px.HttpProxy)
	http.ListenAndServeTLS(":"+httpsPort, certFile, keyFile, px.HttpsProxy)
}

func processRequest(req *http.Request) error {
	log.Println("Request: ", req.Method, req.URL.String())
	//remove gzip from Accept-Encoding to make it easier to parse on Response
	req.Header.Set("Accept-Encoding", "identity")
	return nil
}

func processResponse(resp *http.Response) error {
	var imgExtension string

	// skip not modified or other weird headers that dont include payload
	if resp.StatusCode != 200 {
		return nil
	}

	// skip gzipped assets that ignored us when told no gzip
	if "gzip" == resp.Header.Get("Content-Encoding") {
		return nil
	}

	switch resp.Header.Get("Content-Type") {
	case "image/jpeg":
		imgExtension = ".jpg"
	case "image/png":
		imgExtension = ".png"
	case "image/gif":
		imgExtension = ".gif"
	}

	if "" != imgExtension {
		//log.Println("Processing", resp.Request.Method, resp.Request.URL.String())
		// save locally
		fileName := path.Join(os.TempDir(), strconv.FormatInt(time.Now().UnixNano(), 10)+imgExtension)
		file, _ := os.Create(fileName)
		defer os.Remove(fileName)

		// dump Body to file
		io.Copy(file, resp.Body)
		file.Close()
		resp.Body.Close()

		// detect faces
		image := opencv.LoadImage(fileName, -1)
		defer image.Release()
		if image != nil {
			cascade := opencv.LoadHaarClassifierCascade("./deface_files/haarcascade_frontalface_alt.xml")
			faces := cascade.DetectObjects(image)

			// replace faces
			if len(faces) > 0 {
				rageGuy := opencv.LoadImage("./deface_files/rage-guy.png", -1)
				defer rageGuy.Release()
				for _, value := range faces {
					faceWidth := value.Width()
					faceHeight := value.Height()
					faceX := value.X()
					faceY := value.Y()
					if (faceWidth <= 0) || (faceHeight <= 0) || (faceX > image.Width()) || (faceY > image.Height()) {
						//log.Println("Invalid rectangle for", resp.Request.Method, resp.Request.URL.String(), faceWidth, faceHeight, faceX, faceY)
						continue
					}

					resizedRageGuy := opencv.Resize(rageGuy, faceWidth, faceHeight, 0)
					rect := &opencv.Rect{}
					rect.Init(faceX, faceY, faceWidth, faceHeight)
					image.SetROI(*rect)
					// blending them horribly inefficiently because cvSplit/cvThreshold are not exposed in go
					// and dont want to link to C
					for y := 0; y < resizedRageGuy.Height(); y++ {
						for x := 0; x < resizedRageGuy.Width(); x++ {
							src := image.Get2D(x, y).Val()
							overlay := resizedRageGuy.Get2D(x, y).Val()
							if overlay[3] > 0 {
								image.Set2D(x, y, opencv.NewScalar(overlay[0], overlay[1], overlay[2], src[3]))
							}
						}
					}
					image.ResetROI()
					resizedRageGuy.Release()
				}
			}

			// save modified image and pass it to Body
			opencv.SaveImage(fileName, image, 0)
		} else {
			//log.Println("Couldnt load image", resp.Request.Method, resp.Request.URL.String())
		}

		file, _ = os.Open(fileName)
		resp.Body = ioutil.NopCloser(file)

		// update content length
		fileInfo, _ := file.Stat()
		resp.ContentLength = fileInfo.Size()
		resp.Header.Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	}

	return nil
}
