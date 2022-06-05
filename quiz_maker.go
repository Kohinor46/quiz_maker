package main

import (
	"flag"
	tb "gopkg.in/tucnak/telebot.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

//Тип пользователя
type User struct {
	ID        int64  `yaml:"ID"`
	Points    int    `yaml:"Points"`
	Nick      string `yaml:"Nick"`
	FirstName string `yaml:"FirstName"`
	LastName  string `yaml:"LastName"`
	Help      struct {
		Round_used_fifty_fifty int `yaml:"Round_used_fifty_fifty"` //Когда был использована подсказка 50/50
		//Признаки подсказок
		Fifty_fifty bool `yaml:"Fifty_fifty"`
		Call        bool `yaml:"Call"`
		Statistic   bool `yaml:"Statistic"`
	} `yaml:"Help"`
	Is_round_answers []bool `yaml:"Is_round_answers"` //Признаки ответа на роунды
}

//Тип статистики !!!!Надо переделать под любое количетсво ответов
type Statistic struct {
	One   int `yaml:"One"`
	Two   int `yaml:"Two"`
	Three int `yaml:"Three"`
	Four  int `yaml:"Four"`
}

//Тип конфига
type Config struct {
	Rounds map[int]struct {
		//omitempty - означет, что может отсутсвовать
		With_photo          bool           `yaml:"With_photo,omitempty"`
		Photo_from_disk     bool           `yaml:"Photo_from_disk,omitempty"`
		Photo_from_url      bool           `yaml:"Photo_from_url,omitempty"`
		With_video          bool           `yaml:"With_video,omitempty"`
		Video_from_disk     bool           `yaml:"Video_from_disk,omitempty"`
		Video_from_url      bool           `yaml:"Video_from_url,omitempty"`
		With_audio          bool           `yaml:"With_audio,omitempty"`
		Audio_from_disk     bool           `yaml:"Audio_from_disk,omitempty"`
		Audio_from_url      bool           `yaml:"Audio_from_url,omitempty"`
		Media               string         `yaml:"Media,omitempty"`
		Queston             string         `yaml:"Queston"`
		Answers             map[int]string `yaml:"Answers"`
		Right_answer        int            `yaml:"Right_answer"` // правильный ответ
		Points              int            `yaml:"Points"`
		Fifty_fifty_buttons []int          `yaml:"Fifty_fifty_buttons"` //какие кнопки будут возвращаться при запросе 50/50
	} `yaml:"Rounds"`
	Path_to_files           string  `yaml:"Path_to_files"` //Где хранить файлы с пользователями
	Telegram_token          string  `yaml:"Telegram_token"`
	Welcome                 string  `yaml:"Welcome"` //Текст приветсвия
	Welcome_with_photo      bool    `yaml:"Welcome_with_photo,omitempty"`
	Welcome_photo_from_disk bool    `yaml:"Welcome_photo_from_disk,omitempty"`
	Welcome_photo_from_url  bool    `yaml:"Welcome_photo_from_url,omitempty"`
	Welcome_with_video      bool    `yaml:"Welcome_with_video,omitempty"`
	Welcome_video_from_disk bool    `yaml:"Welcome_video_from_disk,omitempty"`
	Welcome_video_from_url  bool    `yaml:"Welcome_video_from_url,omitempty"`
	Media                   string  `yaml:"Media,omitempty"`
	Admin_ids               []int64 `yaml:"Admin_ids"` //telegram id ведущего/админа который будет запускать раунды
}

//Глобавльные переменные
var (
	currentTime string //текущая дата, обновляется при запуске /start
	config      Config
	b           *tb.Bot
	config_path string
)

//парсим путь до конфиг файла из флага
func init() {
	flag.StringVar(&config_path, "config", "./config.yaml", "path to config file")
}

func main() {
	//читает конфиг файл и парсит его
	yamlFile, err := ioutil.ReadFile(config_path)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Printf("Unmarshal: %v", err)
	}
	currentTime = time.Now().Format("01-02-2006")
	//создает кнопки для админа/ведущего
	buttons_with_rounds_for_admin := [][]tb.InlineButton{}
	result_button := tb.InlineButton{
		Unique: "R",
		Text:   "Результаты",
	}
	//создает массив кнопок с раундами для отправки админу/ведущему
	buttons_with_rounds := make([]tb.InlineButton, len(config.Rounds))
	for idx, _ := range buttons_with_rounds {
		//наполняет данными кнопки
		buttons_with_rounds[idx].Unique = strconv.Itoa(idx + 1) // id+1 - номер раунда
		buttons_with_rounds[idx].Text = strconv.Itoa(idx+1) + " вопрос"
		buttons_with_rounds_for_admin = append(buttons_with_rounds_for_admin, []tb.InlineButton{buttons_with_rounds[idx]})
	}
	buttons_with_rounds_for_admin = append(buttons_with_rounds_for_admin, []tb.InlineButton{result_button})
	//настраивает бота
	b, err = tb.NewBot(tb.Settings{
		Token:  config.Telegram_token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Println("Error connect to bot: ", err)
		return
	}
	//начинает слушать команду /start
	go hadle_start(buttons_with_rounds_for_admin)
	//начинает слушать кнопку с результатми
	go hadle_result(result_button, b)
	for _, button := range buttons_with_rounds_for_admin {
		if button[0].Unique == "R" {
			continue
		}
		//начинает слушать кнопки с раундами от админа/ведущего
		//каждая кнопка слушается в новом потоке
		go hadle_buttons(button)
	}
	//стратует бота
	b.Start()
}

//слушает команду /start
func hadle_start(buttons_with_rounds_for_admin [][]tb.InlineButton) {
	b.Handle("/start", func(m *tb.Message) {
		currentTime = time.Now().Format("01-02-2006")
		//проверяет кто отправил команду
		//если отправил админ то отправятся кнопки с раудами и результатом
		for _, admin := range config.Admin_ids {
			if m.Sender.ID == admin {
				_, err := b.Send(m.Sender, "Меню", &tb.ReplyMarkup{
					InlineKeyboard: buttons_with_rounds_for_admin,
				})
				if err != nil {
					log.Println("Error while send admin keyboard\n", err)
					send_to_admin("Ошибка отправки клавиатуры админа:\n" + err.Error())
				}
				return
			}
		}
		//создает пользователя если его не существует
		adduser(config.Path_to_files+currentTime, m.Sender.ID, m.Sender.Username, m.Sender.FirstName, m.Sender.LastName)
		//подготовка к отправке приветсвенного сообщения
		//с фото
		if config.Welcome_with_photo {
			var a tb.Photo
			//от куда брать фото
			if config.Welcome_photo_from_url {
				a = tb.Photo{File: tb.FromURL(config.Media), Caption: config.Welcome}
			}
			if config.Welcome_photo_from_disk {
				a = tb.Photo{File: tb.FromDisk(config.Media), Caption: config.Welcome}
			}
			//отправляет приветсвенное слово с фото
			_, err := b.Send(m.Chat, &a)
			if err != nil {
				log.Println("Send Welcome with photo: ", err)
				send_to_admin("Ошибка отправки приветсвенного сообщения с фото:\n" + err.Error())
			}
			//с видео
		} else if config.Welcome_with_video {
			var a tb.Video
			//от куда брать видео
			if config.Welcome_video_from_url {
				a = tb.Video{File: tb.FromURL(config.Media), Caption: config.Welcome}
			}
			if config.Welcome_photo_from_disk {
				a = tb.Video{File: tb.FromDisk(config.Media), Caption: config.Welcome}
			}
			//отправляет приветсвенное слово с видео
			_, err := b.Send(m.Chat, &a)
			if err != nil {
				log.Println("Send Welcome with Video: ", err)
				send_to_admin("Ошибка отправки приветсвенного сообщения с видео:\n" + err.Error())
			}
		} else {
			//отправляет приветсвенное слово
			_, err := b.Send(m.Chat, config.Welcome)
			if err != nil {
				log.Println("Send Welcome: ", err)
				send_to_admin("Ошибка отправки приветсвенного сообщения:\n" + err.Error())
			}
		}
	})
}

//слушает кнопку результатов у ведущего/админа
func hadle_result(result tb.InlineButton, b *tb.Bot) {
	b.Handle(&result, func(c *tb.Callback) {
		//получаем всех пользователей
		u := get_users()
		if u == nil {
			/*если пользователей нет то отправлем это сообщение ведущему
			иначе краш с runtime error*/
			_, err := b.Send(c.Sender, "Нет пользователей")
			if err != nil {
				log.Println("Send result: ", err)
				send_to_admin("Ошибка отправки сообщения что нет пользователей:\n" + err.Error())
			}
			err = b.Respond(c, &tb.CallbackResponse{})
			if err != nil {
				log.Println("Respond: ", err)
				send_to_admin("Ошибка ответа по кнопке результатов, что нет пользователей:\n" + err.Error())
			}
			return
		}
		//сортируем пользоватлей по количеству очков
		sort.Slice(u, func(i, j int) bool {
			return u[i].Points > u[j].Points
		})
		//генерируем текст
		var text string
		for idx, x := range u {
			text += strconv.Itoa(idx+1) + " "
			if x.Nick != "" {
				text += "@" + x.Nick + " "
			}
			if x.FirstName != "" {
				text += x.FirstName + " "
			}
			if x.LastName != "" {
				text += x.LastName + "."
			}
			if x.Help.Fifty_fifty {
				text += " Использовал 50/50."
			}
			if x.Help.Call {
				text += " Использовал звонок."
			}
			if x.Help.Statistic {
				text += " Использовал статистику."
			}
			text += " " + strconv.Itoa(x.Points) + "\n"
		}
		//отправляем админу/ведущему
		_, err := b.Send(c.Sender, text)
		if err != nil {
			log.Println("Send result: ", err)
			send_to_admin("Ошибка отправки результатов:\n" + err.Error())
		}
		err = b.Respond(c, &tb.CallbackResponse{})
		if err != nil {
			log.Println("Respond: ", err)
			send_to_admin("Ошибка ответа по кнопке результатов:\n" + err.Error())
		}
	})
}

//Слушает кнопки админа/ведущего
func hadle_buttons(button []tb.InlineButton) {
	b.Handle(&button[0], func(c *tb.Callback) {
		//получает всех пользователей
		u := get_users()
		for idx, _ := range u {
			/*создает чат для отправки вопросов
			потому что нажавший кнопку и получали вопросов разные*/
			se := new(tb.Chat)
			se.ID = u[idx].ID
			//создает кнопки с ответами для отправки
			buttons_with_answers_for_send := [][]tb.InlineButton{}
			round, _ := strconv.Atoi(button[0].Unique)
			for ix, answer := range config.Rounds[round-1].Answers {
				//наполняет кнопки данными
				button_with_answer := tb.InlineButton{
					Unique: button[0].Unique + "_" + strconv.Itoa(ix+1) + "_" + strconv.Itoa(int(se.ID)),
					Text:   strconv.Itoa(ix+1) + ". " + answer,
				}
				//добавляет наполненые данными кнопки в общий массив
				buttons_with_answers_for_send = append(buttons_with_answers_for_send, []tb.InlineButton{button_with_answer})
			}
			//проверяем раунд с видео вопросом
			if config.Rounds[round-1].With_video {
				var a tb.Video
				//проверяем от куда брать видео
				if config.Rounds[round-1].Video_from_url {
					a = tb.Video{File: tb.FromURL(config.Rounds[round-1].Media), Caption: config.Rounds[round-1].Queston}
				}
				if config.Rounds[round-1].Video_from_disk {
					a = tb.Video{File: tb.FromDisk(config.Rounds[round-1].Media), Caption: config.Rounds[round-1].Queston}
				}
				//отправляем вопрос и кнопки с ответами
				_, err := b.Send(se, &a, &tb.ReplyMarkup{
					InlineKeyboard: buttons_with_answers_for_send,
				})
				if err != nil {
					log.Println("Send Queston with Video: ", err)
					send_to_admin("Ошибка отправки раунда #" + button[0].Unique + " c видео:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(se.ID)))
				}
				//проверяем раунд с фото вопросом
			} else if config.Rounds[round-1].With_photo {
				var a tb.Photo
				//проверяем от куда брать фото
				if config.Rounds[round-1].Photo_from_url {
					a = tb.Photo{File: tb.FromURL(config.Rounds[round-1].Media), Caption: config.Rounds[round-1].Queston}
				}
				if config.Rounds[round-1].Photo_from_disk {
					a = tb.Photo{File: tb.FromDisk(config.Rounds[round-1].Media), Caption: config.Rounds[round-1].Queston}
				}
				//отправляем вопрос и кнопки с ответами
				_, err := b.Send(se, &a, &tb.ReplyMarkup{
					InlineKeyboard: buttons_with_answers_for_send,
				})
				if err != nil {
					log.Println("Send Queston with photo: ", err)
					send_to_admin("Ошибка отправки раунда #" + button[0].Unique + " c фото:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(se.ID)))
				}
				//проверяем раунд с аудио вопросом
			} else if config.Rounds[round-1].With_audio {
				var a tb.Audio
				//проверяем от куда брать аудио
				if config.Rounds[round-1].Audio_from_url {
					a = tb.Audio{File: tb.FromURL(config.Rounds[round-1].Media), Caption: config.Rounds[round-1].Queston}
				}
				if config.Rounds[round-1].Audio_from_disk {
					a = tb.Audio{File: tb.FromDisk(config.Rounds[round-1].Media), Caption: config.Rounds[round-1].Queston}
				}
				//отправляем вопрос и кнопки с ответами
				_, err := b.Send(se, &a, &tb.ReplyMarkup{
					InlineKeyboard: buttons_with_answers_for_send,
				})
				if err != nil {
					log.Println("Send Queston with audio: ", err)
					send_to_admin("Ошибка отправки раунда #" + button[0].Unique + " c аудио:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(se.ID)))
				}
				//если предыдущие проверки не прошли то значит раунд текстовый
			} else {
				//отправляем вопрос и кнопки с ответами
				_, err := b.Send(se, config.Rounds[round-1].Queston, &tb.ReplyMarkup{
					InlineKeyboard: buttons_with_answers_for_send,
				})
				if err != nil {
					log.Println("Send to: "+strconv.Itoa(int(se.ID))+" R", round, ": ", err)
					send_to_admin("Ошибка отправки раунда #" + button[0].Unique + ":\n" + err.Error() + "\nКому:" + strconv.Itoa(int(se.ID)))
				}
			}
			helps := []string{"💔", "📞", "📊"}
			//создаем массив кнопок с подсказками для отправки
			help_buttons := [][]tb.InlineButton{}
			for ix, help := range helps {
				//наполянем кнопки данными
				help_button := tb.InlineButton{
					Unique: "Help" + "_" + button[0].Unique + "_" + strconv.Itoa(ix+1) + "_" + strconv.Itoa(int(se.ID)),
					Text:   help,
				}
				//проверяем может ли пользователь использовать подсказку и добавлеям ее в массив для отправки
				switch help {
				case "💔":
					if !u[idx].Help.Fifty_fifty {
						help_buttons = append(help_buttons, []tb.InlineButton{help_button})
					}
				case "📞":
					if !u[idx].Help.Call {
						help_buttons = append(help_buttons, []tb.InlineButton{help_button})
					}
				case "📊":
					if !u[idx].Help.Statistic {
						help_buttons = append(help_buttons, []tb.InlineButton{help_button})
					}
				}
			}
			//если кнопок нет то пропускаем отправку
			if len(helps) == 0 {
				continue
			}
			//отправляем кнопки с подсказками
			_, err := b.Send(se, "Подсказки", &tb.ReplyMarkup{InlineKeyboard: help_buttons})
			if err != nil {
				log.Println("Send helps: ", err)
				send_to_admin("Ошибка отправки подсказок в раунде #" + button[0].Unique + ":\n" + err.Error() + "\nКому:" + strconv.Itoa(int(se.ID)))
			}
			for _, buttons_with_answer := range buttons_with_answers_for_send {
				go hadle_answer_buttons(buttons_with_answer)
			}
			for _, buttons_with_help := range help_buttons {
				go hadle_buttons_with_help(buttons_with_help)
			}
		}
		b.Respond(c, &tb.CallbackResponse{})
	})
}

//слушает кнопки помощи
func hadle_buttons_with_help(bu []tb.InlineButton) {
	b.Handle(&bu[0], func(ca *tb.Callback) {
		//разбивает кнопку на состовные части Help_раунд_ответ_ид пользователя
		an := strings.Split(bu[0].Unique, "_")
		round, _ := strconv.Atoi(an[1])
		u := get_user(an[3])
		if u.Is_round_answers[round-1] {
			b.Respond(ca, &tb.CallbackResponse{Text: "Ты уже ответил"})
			return
		}
		switch bu[0].Text {
		case "💔":
			if u.Help.Fifty_fifty {
				b.Respond(ca, &tb.CallbackResponse{Text: "Подсказка уже использована"})
				return
			}
			buttons_with_answers_for_send := [][]tb.InlineButton{}
			for _, id_button := range config.Rounds[round-1].Fifty_fifty_buttons {
				button_with_answer := tb.InlineButton{
					Unique: an[1] + "_" + an[2] + "_" + an[3],
					Text:   strconv.Itoa(id_button+1) + ". " + config.Rounds[round-1].Answers[id_button],
				}
				buttons_with_answers_for_send = append(buttons_with_answers_for_send, []tb.InlineButton{button_with_answer})
			}
			_, err := b.Send(ca.Sender, config.Rounds[round-1].Queston, &tb.ReplyMarkup{
				InlineKeyboard: buttons_with_answers_for_send,
			})
			u.Help.Fifty_fifty = true
			u.Help.Round_used_fifty_fifty = round - 1
			write_user(u)
			if err != nil {
				log.Println("Send to: "+strconv.Itoa(int(ca.Sender.ID))+" R", round, ": ", err)
				send_to_admin("Ошибка отправки подсказки 50/50:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(ca.Sender.ID)))
			}
		case "📞":
			if u.Help.Call {
				b.Respond(ca, &tb.CallbackResponse{Text: "Подсказка уже использована"})
				return
			}
			var text_for_random []string
			buttons_with_answers_for_send := [][]tb.InlineButton{}
			if round-1 == u.Help.Round_used_fifty_fifty {
				for _, id_button := range config.Rounds[round-1].Fifty_fifty_buttons {
					text_for_random = append(text_for_random, config.Rounds[round-1].Answers[id_button])
					button_with_answer := tb.InlineButton{
						Unique: an[1] + "_" + an[2] + "_" + an[3],
						Text:   strconv.Itoa(id_button+1) + ". " + config.Rounds[round-1].Answers[id_button],
					}
					buttons_with_answers_for_send = append(buttons_with_answers_for_send, []tb.InlineButton{button_with_answer})
				}
				_, err := b.Send(ca.Sender, "Эммм...\nЯ думаю ответ:\n"+choose_random(text_for_random), &tb.ReplyMarkup{
					InlineKeyboard: buttons_with_answers_for_send,
				})
				if err != nil {
					log.Println("Send to: "+strconv.Itoa(int(ca.Sender.ID))+" R", round, ": ", err)
					send_to_admin("Ошибка отправки укороченных подсказкок звонок боту:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(ca.Sender.ID)))
				}
			} else {
				for _, answer := range config.Rounds[round-1].Answers {
					text_for_random = append(text_for_random, answer)
				}
				_, err := b.Send(ca.Sender, "Эммм...\nЯ думаю ответ:\n"+choose_random(text_for_random), &tb.ReplyMarkup{
					InlineKeyboard: buttons_with_answers_for_send,
				})
				if err != nil {
					log.Println("Send to: "+strconv.Itoa(int(ca.Sender.ID))+" R", round, ": ", err)
					send_to_admin("Ошибка отправки подсказки звонок боту:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(ca.Sender.ID)))
				}
			}
			u.Help.Call = true
			write_user(u)
		case "📊":
			if u.Help.Statistic {
				b.Respond(ca, &tb.CallbackResponse{Text: "Подсказка уже использована"})
				return
			}
			s := get_statistic(round - 1)
			var text string
			buttons_with_answers_for_send := [][]tb.InlineButton{}
			if round-1 == u.Help.Round_used_fifty_fifty {
				for _, id_button := range config.Rounds[round-1].Fifty_fifty_buttons {
					button_with_answer := tb.InlineButton{
						Unique: an[1] + "_" + an[2] + "_" + an[3],
						Text:   strconv.Itoa(id_button+1) + ". " + config.Rounds[round-1].Answers[id_button],
					}
					buttons_with_answers_for_send = append(buttons_with_answers_for_send, []tb.InlineButton{button_with_answer})
					switch id_button {
					case 0:
						text += "1-" + strconv.Itoa(s.One) + "\n"
					case 1:
						text += "2-" + strconv.Itoa(s.Two) + "\n"
					case 2:
						text += "3-" + strconv.Itoa(s.Three) + "\n"
					case 3:
						text += "4-" + strconv.Itoa(s.Four) + "\n"
					}
				}
				_, err := b.Send(ca.Sender, text, &tb.ReplyMarkup{InlineKeyboard: buttons_with_answers_for_send})
				if err != nil {
					log.Println("Send to: "+strconv.Itoa(int(ca.Sender.ID))+" R", round, ": ", err)
					send_to_admin("Ошибка отправки укороченной подсказки статистика:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(ca.Sender.ID)))
				}
			} else {
				text = "1-" + strconv.Itoa(s.One) + "\n2-" + strconv.Itoa(s.Two) + "\n3-" + strconv.Itoa(s.Three) + "\n4-" + strconv.Itoa(s.Four)
				_, err := b.Send(ca.Sender, text, &tb.ReplyMarkup{InlineKeyboard: buttons_with_answers_for_send})
				if err != nil {
					log.Println("Send to: "+strconv.Itoa(int(ca.Sender.ID))+" R", round, ": ", err)
					send_to_admin("Ошибка отправки подсказки статистика:\n" + err.Error() + "\nКому:" + strconv.Itoa(int(ca.Sender.ID)))
				}
			}
			u.Help.Statistic = true
			write_user(u)
		}
		b.Respond(ca, &tb.CallbackResponse{})
	})
}

//Слушаем кнопку с ответами
func hadle_answer_buttons(bu []tb.InlineButton) {
	b.Handle(&bu[0], func(ca *tb.Callback) {
		//разбиваем кнопку на составные части раунд_ответ_ид пользователя
		an := strings.Split(bu[0].Unique, "_")
		answer, _ := strconv.Atoi(an[1])
		round, _ := strconv.Atoi(an[0])
		u := get_user(an[2])
		if u.Is_round_answers[round-1] {
			b.Respond(ca, &tb.CallbackResponse{Text: "Ты уже отвечал"})
			return
		}
		if config.Rounds[round-1].Right_answer == answer-1 {
			add_points(an[2], config.Rounds[round-1].Points)
		}
		write_answer(round-1, an[2])
		if round != 15 {
			write_statistic(round-1, answer-1)
		}
		b.Respond(ca, &tb.CallbackResponse{Text: "Ответ принят"})
	})
}

//Возвращает случайную строчку из списка строк
func choose_random(reasons []string) string {
	rand.Seed(time.Now().Unix())
	n := rand.Int() % len(reasons)
	return reasons[n]
}

//Возвращает статистику по раунду
func get_statistic(r int) Statistic {
	yamlFile, err := ioutil.ReadFile(config.Path_to_files + "statistic.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		send_to_admin("Ошибка чтения файла статистики при получении статистики\n:" + err.Error())
	}
	var s []Statistic
	err = yaml.Unmarshal(yamlFile, &s)
	if err != nil {
		log.Printf("Unmarshal: %v", err)
		send_to_admin("Ошибка парсинга файла статистики при получении статистики:\n" + err.Error())
	}
	return s[r]
}

//Добавляет в файл статистики информацию о нажатии пользователей
func write_statistic(r int, answers int) {
	//Проверка на существование файла статистики
	if _, err := os.Stat(config.Path_to_files + "statistic.yaml"); os.IsNotExist(err) {
		//если файл не существует то создаем его и наполняем пустыми данными
		f, _ := os.Create(config.Path_to_files + "statistic.yaml")
		f.Close()
		s := make([]Statistic, len(config.Rounds))
		content, err := yaml.Marshal(s)
		if err != nil {
			log.Printf("Marshal: %v", err)
			send_to_admin("Ошибка превращения пустой структуры в yaml статистики:\n" + err.Error())
		}
		err = ioutil.WriteFile(config.Path_to_files+"statistic.yaml", content, 0666)
		if err != nil {
			log.Println("WriteFile: ", err)
			send_to_admin("Ошибка записи в файл статистики пустого значения:\n" + err.Error())
		}
	}
	//Читаем файл
	yamlFile, err := ioutil.ReadFile(config.Path_to_files + "statistic.yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		send_to_admin("Ошибка чтения файла статистики при записи в статистику:\n" + err.Error())
	}
	var s []Statistic
	err = yaml.Unmarshal(yamlFile, &s)
	if err != nil {
		log.Printf("Unmarshal: %v", err)
		send_to_admin("Ошибка парсинга файла статистики при записи в статистику:\n" + err.Error())
	}
	u := &s[r]
	//Добавляем информацию по нажатию  !!!! Переделать на любое количество ответов
	switch answers {
	case 0:
		u.One++
	case 1:
		u.Two++
	case 2:
		u.Three++
	case 3:
		u.Four++
	}
	content, err := yaml.Marshal(s)
	if err != nil {
		log.Printf("Marshal: %v", err)
		send_to_admin("Ошибка превращения структуры в yaml статистики:\n" + err.Error())
	}
	err = ioutil.WriteFile(config.Path_to_files+"statistic.yaml", content, 0666)
	if err != nil {
		log.Println("WriteFile: ", err)
		send_to_admin("Ошибка записи в файл статистики значений:\n" + err.Error())
	}
}

//Записывает признак ответа
func write_answer(round int, id string) {
	u := get_user(id)
	u.Is_round_answers[round] = true
	write_user(u)
}

//Добавляет очки пользователю
func add_points(id string, p int) {
	u := get_user(id)
	u.Points += p
	write_user(u)
}

//Возвращает одного пользователя
func get_user(file_name string) User {
	yamlFile, err := ioutil.ReadFile(config.Path_to_files + currentTime + "/" + file_name + ".yaml")
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
		send_to_admin("Ошибка чтения файла пользователя" + file_name + ":\n" + err.Error())
	}
	var u User
	err = yaml.Unmarshal(yamlFile, &u)
	if err != nil {
		log.Printf("Unmarshal: %v", err)
		send_to_admin("Ошибка парсинга пользователя" + file_name + ":\n" + err.Error())
	}
	return u
}

//Возвращает всех пользователей из директории
func get_users() []User {
	if _, err := os.Stat(config.Path_to_files + currentTime); os.IsNotExist(err) {
		return nil
	}
	files, err := ioutil.ReadDir(config.Path_to_files + currentTime)
	if err != nil {
		log.Println("Error read dir: ", err)
		send_to_admin("Ошибка чтения директории с пользователями:\n" + err.Error())
	}
	var users []User
	for _, f := range files {
		u := get_user(f.Name())
		users = append(users, u)
	}
	return users
}

//Проверяет и создает пользователя, файл и папки
func adduser(dir string, user int64, nick string, FirstName string, LastName string) {
	if _, err := os.Stat(config.Path_to_files + currentTime); os.IsNotExist(err) {
		os.Mkdir(config.Path_to_files+currentTime, 0755)
	}
	if _, err := os.Stat(config.Path_to_files + currentTime + "/" + strconv.Itoa(int(user)) + ".yaml"); os.IsNotExist(err) {
		f, err := os.Create(config.Path_to_files + currentTime + "/" + strconv.Itoa(int(user)) + ".yaml")
		f.Close()
		if err != nil {
			log.Println("Can't create user file: ", err)
			send_to_admin("Ошибка создания файла пользователя" + strconv.Itoa(int(user)) + ":\n" + err.Error())
		}
	} else if err == nil {
		return
	}
	var a User
	a.ID = user
	a.Nick = nick
	a.FirstName = FirstName
	a.LastName = LastName
	a.Help.Round_used_fifty_fifty = -1
	a.Is_round_answers = make([]bool, len(config.Rounds))
	write_user(a)
}

//Запись пользователя в файл
func write_user(u User) {
	content, err := yaml.Marshal(u)
	if err != nil {
		log.Printf("Marshal: %v", err)
		send_to_admin("Ошибка превращения пользователя " + strconv.Itoa(int(u.ID)) + " в структуру для записи в файл:\n" + err.Error())
	}
	err = ioutil.WriteFile(config.Path_to_files+currentTime+"/"+strconv.Itoa(int(u.ID))+".yaml", content, 0666)
	if err != nil {
		log.Println("WriteFile: ", err)
		send_to_admin("Ошибка записи пользователя " + strconv.Itoa(int(u.ID)) + " в файл:\n" + err.Error())
	}
}

func send_to_admin(text string) {
	if len(text) > 4095 {
		for i := 0; i < len(text); i += 4095 {
			for _, admin_id := range config.Admin_ids {
				se := new(tb.Chat)
				se.ID = admin_id
				if i+4095 > len(text) {
					_, err := b.Send(se, text[i:len(text)])
					if err != nil {
						log.Println("Error send error to admin: ", err)
						send_to_admin("Ошибка отправки ошибки:\n" + err.Error())
					}
				} else {
					_, err := b.Send(se, text[i:i+4095])
					if err != nil {
						log.Println("Error send error to admin: ", err)
						send_to_admin("Ошибка отправки ошибки:\n" + err.Error())
					}
				}
			}
		}
	} else {
		for _, admin_id := range config.Admin_ids {
			se := new(tb.Chat)
			se.ID = admin_id
			_, err := b.Send(se, text)
			if err != nil {
				log.Println("Error send error to admin: ", err)
				send_to_admin("Ошибка отправки ошибки:\n" + err.Error())
			}
		}
	}
}
