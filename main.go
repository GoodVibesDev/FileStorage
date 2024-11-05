package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func uploadFile(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	//added to enable sending process monitoring
	w.WriteHeader(http.StatusOK)

	// If file size > 45Mb, then throw error
	if r.ContentLength > 45<<20 {
		http.Error(w, "File is too large", http.StatusRequestEntityTooLarge)
		return
	}

	//45mb - max uploadedFile size
	err := r.ParseMultipartForm(45 << 20)

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

func downloadFile(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	//Getting path variable to find the file
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

	//Remove folders from path and timestamp
	fileName := getFileName(file.Name())

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName+"\"")
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
	r.HandleFunc("/download/{fileName}", downloadFile)

	http.ListenAndServe(":8085", r)
}

func main() {
	//Creates dir for files storing
	os.Mkdir("files", os.ModePerm)
	setupRoutes()
}
