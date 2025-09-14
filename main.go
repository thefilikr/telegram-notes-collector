package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Config struct {
	Token  string `json:"token"`
	UserID int64  `json:"user_id"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func main() {
	configPath := "config.json"

	var cfg *Config
	var err error

	// Загружаем или создаём конфиг
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		var token string
		var userID int64

		fmt.Print("Введите токен бота: ")
		fmt.Scanln(&token)

		fmt.Print("Введите ваш Telegram ID: ")
		fmt.Scanln(&userID)

		cfg = &Config{Token: token, UserID: userID}
		if err := saveConfig(configPath, cfg); err != nil {
			fmt.Println("Ошибка сохранения конфига:", err)
			return
		}
	} else {
		cfg, err = loadConfig(configPath)
		if err != nil {
			fmt.Println("Ошибка загрузки конфига:", err)
			return
		}
	}

	// Спрашиваем путь для заметок
	var notesDir string
	fmt.Print("Введите путь к папке для заметок: ")
	fmt.Scanln(&notesDir)
	if err := os.MkdirAll(notesDir, 0755); err != nil {
		fmt.Println("Ошибка создания папки:", err)
		return
	}

	// Инициализация бота
	bot, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		fmt.Println("Ошибка создания бота:", err)
		return
	}

	fmt.Println("Бот запущен, ждём сообщения...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// фильтрация по ID
		if update.Message.From.ID != cfg.UserID {
			continue
		}

		// Формируем имя файла
		filename := fmt.Sprintf("%d_%d.md", update.UpdateID, time.Now().Unix())
		filepathMD := filepath.Join(notesDir, filename)

		// Сохраняем текст сообщения
		content := fmt.Sprintf("# Сообщение от %s\n\n%s\n\n", update.Message.From.UserName, update.Message.Text)

		// Обработка файлов (документы, фото)
		if update.Message.Document != nil {
			file := update.Message.Document
			url, _ := bot.GetFileDirectURL(file.FileID)
			localFile := filepath.Join(notesDir, file.FileName)
			if err := downloadFile(localFile, url); err == nil {
				content += fmt.Sprintf("📎 Вложение: [%s](%s)\n", file.FileName, file.FileName)
			}
		}
		if len(update.Message.Photo) > 0 {
			photo := update.Message.Photo[len(update.Message.Photo)-1]
			url, _ := bot.GetFileDirectURL(photo.FileID)
			localFile := filepath.Join(notesDir, fmt.Sprintf("photo_%d.jpg", time.Now().Unix()))
			if err := downloadFile(localFile, url); err == nil {
				content += fmt.Sprintf("🖼 Фото: ![](%s)\n", filepath.Base(localFile))
			}
		}

		// Записываем в md файл
		if err := os.WriteFile(filepathMD, []byte(content), 0644); err != nil {
			fmt.Println("Ошибка записи заметки:", err)
			continue
		}
		fmt.Println("Сохранена заметка:", filepathMD)
	}
}

func downloadFile(path, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
