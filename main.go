package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ContainerStatus struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func main() {
	// Создание Docker клиента
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("Ошибка при создании Docker клиента: %v", err)
	}

	r := mux.NewRouter()

	// Маршрут для веб-сокетов
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer ws.Close()

		for {
			// Получение списка контейнеров через Docker API
			containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
			if err != nil {
				log.Println("Ошибка при получении списка контейнеров:", err)
				return
			}

			// Формирование данных для отправки
			var statusList []ContainerStatus
			for _, container := range containers {
				statusList = append(statusList, ContainerStatus{
					ID:     container.ID[:12],      // Берем первые 12 символов ID
					Name:   container.Names[0][1:], // Убираем первый символ "/"
					Status: container.State,
				})
			}

			// Отправка данных через WebSocket
			err = ws.WriteJSON(statusList)
			if err != nil {
				log.Println("Ошибка при отправке данных через WebSocket:", err)
				return
			}

			// Пауза перед следующим обновлением
			time.Sleep(2 * time.Second)
		}
	})

	// Маршрут для отображения HTML-страницы
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	// Запуск сервера
	log.Println("Сервер запущен на :1111")
	log.Fatal(http.ListenAndServe(":1111", r))
}
