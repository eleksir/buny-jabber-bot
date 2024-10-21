package jabber

import (
	"time"

	log "github.com/sirupsen/logrus"
)

// MyLoop - основной цикл программы.
func (j *Jabber) MyLoop() {
	for {
		select {
		case <-j.GTomb.Dying():
			j.GTomb.Done()
			return //nolint:nlreturn

		default:
			// Зададим начальное значение глобальным переменным
			j.ServerPingTimestampRx = 0
			j.ServerPingTimestampTx = 0
			j.RoomsConnected = make([]string, 1)
			j.LastActivity = 0
			j.LastServerActivity = 0
			j.LastMucActivity = NewCollection()
			j.ServerCapsQueried = false
			j.ServerCapsList = NewCollection()
			j.MucCapsList = NewCollection()
			j.ServerPingTimestampTx = 0
			j.ServerPingTimestampRx = 0
			j.RoomPresences = NewCollection()

			// Установим коннект
			j.EstablishConnection()

			j.ServerPingTimestampRx = time.Now().Unix() // Считаем, что если коннект запустился, то первый пинг успешен.

			// Тыкаем сервер палочкой, проверяем, что коннект жив и вываливаемся из mainLoop, если он не жив.
			go j.ProbeServerLiveness()

			// Тыкаем muc-и палочкой, проверяем, что они живы и вываливаемся из mainLoop, если пинги пропали.
			// Если пинги до комнаты пропали, то это фактически значит, что либо сервер потерял связь с MUC-компонентом,
			// либо у нас какой-то wire error.
			go j.ProbeMUCLiveness()

			// Гребём ивенты...
			for {
				// Стриггерилось завершение работы приложения, или соединение не установлено (порвалось, например)
				// грести не надо
				if j.Shutdown {
					break
				}

				if !j.IsConnected {
					// Tight loop - это наверно не очень хорошо, думаю, ничего страшного не будет, если мы поспим 100мс.
					time.Sleep(100 * time.Millisecond)

					continue
				}

				// Вынимаем ивент из "провода"
				chat, err := j.Talk.Recv()

				if err != nil {
					log.Errorf("Unable to get events from server: %s", err)
					j.GTomb.Kill(err)

					// Не забываем выходить на ошибке :(
					return
				}

				j.ParseEvent(chat)
			}
		}
	}
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
