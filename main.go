package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

//go:embed web/*
var assets embed.FS

const version = "1.0.0"

func localAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast())
}

func loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func lanURLs(port int) []string {
	urls := []string{}
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err == nil && ip.To4() != nil && (ip.IsPrivate() || ip.IsLinkLocalUnicast()) {
				urls = append(urls, fmt.Sprintf("http://%s:%d", ip.String(), port))
			}
		}
	}
	sort.Strings(urls)
	return urls
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newWebHandler(app *App, port int, shutdown func()) http.Handler {
	web, _ := fs.Sub(assets, "web")
	allowedHosts := map[string]bool{fmt.Sprintf("127.0.0.1:%d", port): true, fmt.Sprintf("localhost:%d", port): true}
	for _, address := range lanURLs(port) {
		u, _ := url.Parse(address)
		allowedHosts[u.Host] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		if !localAddress(r.RemoteAddr) {
			jsonResponse(w, 403, map[string]string{"error": "Esta app solo admite conexiones de la red local."})
			return
		}
		hostAllowed := allowedHosts[strings.ToLower(r.Host)]
		if !hostAllowed {
			// Wi-Fi reconnection and DHCP may add an address after the app starts.
			for _, address := range lanURLs(port) {
				u, _ := url.Parse(address)
				if strings.EqualFold(u.Host, r.Host) {
					hostAllowed = true
					break
				}
			}
		}
		if !hostAllowed {
			jsonResponse(w, 403, map[string]string{"error": "Usa la dirección local que muestra el anfitrión."})
			return
		}
		switch r.URL.Path {
		case "/api/info":
			if r.Method != http.MethodGet {
				jsonResponse(w, 405, map[string]string{"error": "Método no permitido."})
				return
			}
			jsonResponse(w, 200, map[string]any{"name": "Nexo Chess", "version": version, "urls": lanURLs(port), "isHost": loopback(r.RemoteAddr)})
		case "/api/shutdown":
			if r.Method != http.MethodPost {
				jsonResponse(w, 405, map[string]string{"error": "Método no permitido."})
				return
			}
			if !loopback(r.RemoteAddr) || !mutationAllowed(r) || !app.HasToken(bearerToken(r)) {
				jsonResponse(w, 403, map[string]string{"error": "Solo el anfitrión puede apagar el servidor."})
				return
			}
			var body struct{}
			if decodeBody(w, r, &body) != nil {
				jsonResponse(w, 400, map[string]string{"error": "Se requiere un objeto JSON válido."})
				return
			}
			jsonResponse(w, 200, map[string]bool{"ok": true})
			if shutdown != nil {
				go shutdown()
			}
		default:
			if strings.HasPrefix(r.URL.Path, "/api/") {
				app.ServeHTTP(w, r)
				return
			}
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "Método no permitido", 405)
				return
			}
			path := strings.TrimPrefix(r.URL.Path, "/")
			if path == "" {
				path = "index.html"
			}
			// Serve only exact embedded files; no directories, disk access or SPA fallback.
			if strings.Contains(path, "/") || strings.Contains(path, "\\") || strings.Contains(path, "..") {
				http.NotFound(w, r)
				return
			}
			body, err := fs.ReadFile(web, path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			switch {
			case strings.HasSuffix(path, ".html"):
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			case strings.HasSuffix(path, ".css"):
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			case strings.HasSuffix(path, ".js"):
				w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
			case strings.HasSuffix(path, ".svg"):
				w.Header().Set("Content-Type", "image/svg+xml")
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			if r.Method == http.MethodGet {
				_, _ = w.Write(body)
			}
		}
	})
}

func openBrowser(address string) {
	shell := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	verb, _ := syscall.UTF16PtrFromString("open")
	target, _ := syscall.UTF16PtrFromString(address)
	result, _, _ := shell.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, 1)
	if result <= 32 {
		fmt.Println("No se pudo abrir el navegador. Abre esta dirección:", address)
	}
}

func main() {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	_, _, _ = kernel.NewProc("SetConsoleOutputCP").Call(65001)
	title, _ := syscall.UTF16PtrFromString("Nexo Chess · Ajedrez en red")
	_, _, _ = kernel.NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(title)))
	noBrowser := flag.Bool("no-browser", false, "No abrir el navegador")
	portFlag := flag.Int("port", 0, "Puerto específico (predeterminado: 8088 o el siguiente disponible)")
	showVersion := flag.Bool("version", false, "Mostrar versión")
	flag.Parse()
	if *showVersion {
		fmt.Println("Nexo Chess", version)
		return
	}
	start := 8088
	attempts := 11
	if *portFlag != 0 {
		start = *portFlag
		attempts = 1
	}
	if start < 1024 || start > 65535 {
		fmt.Println("El puerto debe estar entre 1024 y 65535.")
		os.Exit(1)
	}
	var listener net.Listener
	var err error
	port := start
	for i := 0; i < attempts; i++ {
		port = start + i
		listener, err = net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", port))
		if err == nil {
			break
		}
	}
	if err != nil {
		fmt.Println("No se pudo abrir el servidor:", err)
		fmt.Println("Prueba otro puerto con NexoChess.exe --port 8099")
		os.Exit(1)
	}
	defer listener.Close()
	app := NewApp()
	stop := make(chan struct{}, 1)
	shutdown := func() {
		select {
		case stop <- struct{}{}:
		default:
		}
	}
	server := &http.Server{Handler: newWebHandler(app, port, shutdown), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	local := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Print("\n  NEXO CHESS · AJEDREZ EN RED\n\n")
	fmt.Println("  En este ordenador:", local)
	urls := lanURLs(port)
	if len(urls) == 0 {
		fmt.Println("  No hay conexión LAN detectada. Conéctate al Wi-Fi o al cable de red.")
	}
	for _, address := range urls {
		fmt.Println("  Comparte con el otro equipo:", address)
	}
	fmt.Println("\n  Ambos equipos deben estar en la misma red.")
	fmt.Println("  Si Windows lo solicita, permite acceso en redes privadas.")
	fmt.Println("  Mantén esta ventana abierta. Ctrl+C o 'Apagar servidor' para cerrar.")
	fmt.Print("  Las partidas se borran al cerrar; puedes exportar PGN en la web.\n\n")
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			fmt.Println("Error de servidor:", serveErr)
			shutdown()
		}
	}()
	if !*noBrowser {
		openBrowser(local)
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)
	select {
	case <-sig:
	case <-stop:
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	fmt.Println("Servidor cerrado. Las partidas temporales se han eliminado.")
}
