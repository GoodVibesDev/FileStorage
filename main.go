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
	"time"
)

func uploadFile(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	//32mb - max uploadedFile size
	err := r.ParseMultipartForm(50 << 20)

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uploadedFile, handler, err := r.FormFile("file")
	defer uploadedFile.Close()

	if err != nil {
		fmt.Println(err)
		return
	}

	newFile, err := os.Create("files/" + strconv.FormatInt(time.Now().Unix(), 10) + "_" + handler.Filename)
	defer newFile.Close()

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(newFile, uploadedFile); err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(filepath.Base(newFile.Name())))
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	vars := mux.Vars(r)
	fileName := vars["fileName"]

	file, err := os.Open("files/" + fileName)
	if err != nil {
		http.Error(w, "File not found.", http.StatusNotFound)
		return
	}
	defer file.Close()

	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream" // Fallback for unknown types
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(file.Name())+"\"")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	w.Header().Set("Content-Type", contentType)

	http.ServeContent(w, r, file.Name(), time.Now(), file)
}

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
	setupRoutes()
}
