package services

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mslotwinski-dev/dash/src/backend"
	"github.com/mslotwinski-dev/dash/src/utils"
)

// Definiujemy strukturę danych, jaką wyślemy do panelu
type DashboardStats struct {
	TotalRequests  uint64   `json:"total_requests"`
	ActiveBackends []string `json:"active_backends"`
	DeadBackends   []string `json:"dead_backends"`
	LastRequest    string   `json:"last_request"`
	CPUUsage       float64  `json:"cpu_usage"`
	RAMUsage       float64  `json:"ram_usage"`
}

// Hub zarządza wszystkimi aktywnymi połączeniami WebSocket
type WsHub struct {
	clients   map[*websocket.Conn]bool
	Broadcast chan DashboardStats // <--- MUSI BYĆ DUŻA LITERA "B"
	mu        sync.Mutex
}

func NewWsHub() *WsHub {
	return &WsHub{
		clients:   make(map[*websocket.Conn]bool),
		Broadcast: make(chan DashboardStats, 10),
	}
}

// Uruchamia pętlę, która czeka na nowe statystyki i rozsyła je do przeglądarek
func (h *WsHub) Start() {
	go func() {
		for stats := range h.Broadcast {
			h.mu.Lock()
			for client := range h.clients {
				data, _ := json.Marshal(stats)
				err := client.WriteMessage(websocket.TextMessage, data)
				if err != nil {
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}()
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // Zezwalamy na połączenia z każdego źródła
}

func (h *WsHub) HandleWS(w http.ResponseWriter, r *http.Request, getBackends func() []*backend.Backend) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		utils.Error("Błąd upgrade do WebSocket: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = true
	h.mu.Unlock()

	utils.Info("[WS] Nowy administrator połączył się z panelem.")
}

// Stała z wbudowanym HTML, żeby dashboard tworzył się automatycznie
const DashboardHTML = `
<!DOCTYPE html>
<html lang="pl">
<head>
    <meta charset="UTF-8">
    <title>Dash - Panel Administracyjny</title>
    <style>
        body { font-family: 'Segoe UI', sans-serif; background: #1e1e24; color: #fff; padding: 20px; }
        .card { background: #2a2a35; padding: 20px; margin: 10px 0; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.3); }
        h1 { color: #4caf50; }
        .status-up { color: #4caf50; font-weight: bold; }
        .status-down { color: #f44336; font-weight: bold; }
    </style>
</head>
<body>
    <h1>🚀 Dash Live Dashboard</h1>
    <div class="card">
        <h2>Statystyki ogólne</h2>
        <p>Wszystkie żądania od startu serwera (z Prometheusa): <span id="total-requests" style="font-size: 24px; color: #2196f3;">0</span></p>
        <p>Ostatnie żądanie: <span id="last-request" style="color: #ffeb3b;">Brak</span></p>
    </div>
    <div class="card">
        <h2>Stan serwera</h2>
        <p>CPU: <span id="cpu-usage" style="color: #ff9800;">0%</span></p>
        <p>RAM: <span id="ram-usage" style="color: #ff9800;">0%</span></p>
    </div>
    <div class="card">
        <h2>Stan infrastruktury (Backendy)</h2>
        <h3>Działające (Active):</h3>
        <ul id="active-list" class="status-up"></ul>
        <h3>Awaria (Dead):</h3>
        <ul id="dead-list" class="status-down"></ul>
    </div>
    <script>
        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        wsUrl.protocol = protocol;
        const ws = new WebSocket(wsUrl.toString());
        ws.onmessage = function(event) {
            const stats = JSON.parse(event.data);
            document.getElementById("total-requests").innerText = stats.total_requests;
            document.getElementById("last-request").innerText = stats.last_request;
            document.getElementById("cpu-usage").innerText = stats.cpu_usage.toFixed(2) + "%";
            document.getElementById("ram-usage").innerText = stats.ram_usage.toFixed(2) + "%";
            
            const activeList = document.getElementById("active-list");
            activeList.innerHTML = "";
            if(stats.active_backends) {
                stats.active_backends.forEach(url => { activeList.innerHTML += "<li>🟢 " + url + "</li>"; });
            }
            const deadList = document.getElementById("dead-list");
            deadList.innerHTML = "";
            if(stats.dead_backends) {
                stats.dead_backends.forEach(url => { deadList.innerHTML += "<li>🔴 " + url + "</li>"; });
            }
        };
        ws.onclose = function() { console.log("Połączenie z Dash zostało zamknięte."); };
    </script>
</body>
        const wsUrl = new URL("ws", window.location.href);
`
