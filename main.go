package main

import (
  "encoding/json"
    "bytes"
	"fmt"
	"github.com/gorilla/mux"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type UrlFileUploadData struct {
  Url string `json:"url"`
  FileName string `json:"fileName"`
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	//added to enable sending process monitoring
	w.WriteHeader(http.StatusOK)

	maxSize := int64(45 << 20)	// 45mb - max uploadedFile size

	if r.ContentLength > maxSize {
		http.Error(w, "File is too large", http.StatusRequestEntityTooLarge)
		return
	}

	err := r.ParseMultipartForm(maxSize)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	uploadedFile, handler, err := r.FormFile("file")
	defer uploadedFile.Close()

	if err != nil {
		fmt.Println(err)
		return
	}

	//creating file with unix timestamp in name
	newFile, err := os.Create("files/" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + handler.Filename)
	defer newFile.Close()

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//saving data to created file
	if _, err := io.Copy(newFile, uploadedFile); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(filepath.Base(newFile.Name())))
}

// Reads file from url in request and saves it to storage
func uploadFileFromUrl(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

  // Create a decoder for JSON
  decoder := json.NewDecoder(r.Body)
  var data UrlFileUploadData
  err := decoder.Decode(&data)
	if err != nil {
	  fmt.Println(err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

	defer r.Body.Close()

  fileUrl := data.Url
	fileName := data.FileName

  storeFileName := ""
	if r.URL.Query().Get("no_timestamp") == "true" {
	  storeFileName = "files/" + fileName
	} else {
	  storeFileName = "files/" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + fileName
	}

	// creating file with unix timestamp in name
	newFile, err := os.Create(storeFileName)
	defer newFile.Close()

	if err != nil {
	  fmt.Println(err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  // Download file from url
  resp, err := http.Get(fileUrl)
  if err != nil {
    fmt.Println(err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  defer resp.Body.Close()

  var buf bytes.Buffer
  teeReader := io.TeeReader(resp.Body, &buf)
  body, err := io.ReadAll(teeReader)
  if err != nil {
    fmt.Println(err)
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }

  // Detect image type
  _, format, err := image.DecodeConfig(bytes.NewReader(body))
  if err != nil {
    fmt.Println(err)
    http.Error(w, "Invalid image format", http.StatusBadRequest)
    return
  }

  // Decode the image
  img, _, err := image.Decode(bytes.NewReader(body))
  if err != nil {
    fmt.Println(err)
    http.Error(w, "Error decoding image", http.StatusInternalServerError)
    return
  }

  // Compress the image based on format
  switch format {
  case "jpeg":
    err = jpeg.Encode(newFile, img, &jpeg.Options{Quality: 75})
  case "png":
    // Compress as PNG
    err = png.Encode(newFile, img)
  default:
    // If not JPEG or PNG, copy the original file
    resp.Body = io.NopCloser(io.TeeReader(resp.Body, &bytes.Buffer{}))
    _, err = io.Copy(newFile, resp.Body)
  }

  if err != nil {
    fmt.Println(err)
    http.Error(w, "Error compressing image", http.StatusInternalServerError)
    return
  }

	w.Write([]byte(filepath.Base(newFile.Name())))
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	// Getting path variable to find the file
	vars := mux.Vars(r)
	storageFileName := vars["fileName"]

	file, err := os.Open("files/" + storageFileName)
	if err != nil {
		http.Error(w, "File not found.", http.StatusNotFound)
		return
	}
	defer file.Close()

	ext := filepath.Ext(storageFileName)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream" // Fallback for unknown types
	}

	// Remove folders from path and timestamp
	fileName := getFileName(file.Name())

	w.Header().Set("Content-Disposition", "attachment; filename=\"" + fileName + "\"")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	w.Header().Set("Content-Type", contentType)

	http.ServeContent(w, r, file.Name(), time.Now(), file)
}

// Removing timestamp from file name
func getFileName(filePath string) string {
	fileName := filepath.Base(filePath)
	if index := strings.Index(fileName, "_"); index != -1 {
		return fileName[index+1:]
	}
	return fileName
}

// Method to enable cors for specified response
func enableCors(w *http.ResponseWriter) {
	//For now enabling all
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func setupRoutes() {
	r := mux.NewRouter()

	r.HandleFunc("/upload", uploadFile)
	r.HandleFunc("/uploadFromUrl", uploadFileFromUrl)
	r.HandleFunc("/download/{fileName}", downloadFile)

	http.ListenAndServe(":8085", r)
}

func main() {
	//Creates dir for files storing
	os.Mkdir("files", os.ModePerm)
	setupRoutes()
}
