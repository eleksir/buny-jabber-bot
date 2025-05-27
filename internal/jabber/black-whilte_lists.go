package jabber

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hjson/hjson-go"
	log "github.com/sirupsen/logrus"
)

// ReadWhitelist читает и валидирует белые списки пользователей.
func (j *Jabber) ReadWhitelist() error {
	var (
		whitelistLoaded = false
		err             error
		executablePath  string
	)

	executablePath, err = os.Executable()

	if err != nil {
		err = fmt.Errorf("unable to get current executable path: %w", err)

		return err
	}

	whitelistJSONPath := fmt.Sprintf("%s/data/whitelist.json", filepath.Dir(executablePath))

	locations := []string{
		"~/.buny-jabber-bot-whitelist.json",
		"~/buny-jabber-bot-whitelist.json",
		"/etc/buny-jabber-bot-whitelist.json",
		whitelistJSONPath,
	}

	for _, location := range locations {
		fileInfo, err := os.Stat(location)

		// Предполагаем, что файла либо нет, либо мы не можем его прочитать, второе надо бы логгировать, но пока забьём
		if err != nil {
			continue
		}

		// Файл белого списка длинноват для белого списка, попробуем следующего кандидата
		if fileInfo.Size() > 2097152 {
			log.Warnf("Whitelist file %s is too long for whitelist, skipping", location)

			continue
		}

		buf, err := os.ReadFile(location)

		// Не удалось прочитать, попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip reading whitelist file %s: %s", location, err)

			continue
		}

		// Исходя из документации, hjson какбы умеет парсить "кривой" json, но парсит его в map-ку.
		// Интереснее на выходе получить структурку: то есть мы вначале конфиг преобразуем в map-ку, затем эту map-ку
		// сериализуем в json, а потом json превращаем в структурку. Не очень эффективно, но он и нечасто требуется.
		var (
			sampleWhitelist MyWhiteList
			tmp             map[string]interface{}
		)

		err = hjson.Unmarshal(buf, &tmp)

		// Не удалось распарсить - попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		tmpJSON, err := json.Marshal(tmp)

		// Не удалось преобразовать map-ку в json
		if err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		if err := json.Unmarshal(tmpJSON, &sampleWhitelist); err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		j.WhiteList = sampleWhitelist
		whitelistLoaded = true

		log.Infof("Using %s as whiteList file", location)

		break
	}

	if !whitelistLoaded {
		return errors.New("whitelist was not loaded") //nolint:goerr113
	}

	return err
}

// ReadBlacklist читает и валидирует чёрные списки пользователей.
func (j *Jabber) ReadBlacklist() error {
	var (
		blacklistLoaded = false
		err             error
		executablePath  string
	)

	executablePath, err = os.Executable()

	if err != nil {
		err = fmt.Errorf("unable to get current executable path: %w", err)

		return err
	}

	whitelistJSONPath := fmt.Sprintf("%s/data/blacklist.json", filepath.Dir(executablePath))

	locations := []string{
		"~/.bunyPresense-jabber-bot-blacklist.json",
		"~/bunyPresense-jabber-bot-blacklist.json",
		"/etc/bunyPresense-jabber-bot-blacklist.json",
		whitelistJSONPath,
	}

	for _, location := range locations {
		fileInfo, err := os.Stat(location)

		// Предполагаем, что файла либо нет, либо мы не можем его прочитать, второе надо бы логгировать, но пока забьём
		if err != nil {
			continue
		}

		// Файл чёрного списка длинноват для чёрного списка, попробуем следующего кандидата
		if fileInfo.Size() > 16777216 {
			log.Warnf("Blacklist file %s is too long for blacklist, skipping", location)

			continue
		}

		buf, err := os.ReadFile(location)

		// Не удалось прочитать, попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip reading blacklist file %s: %s", location, err)

			continue
		}

		// Исходя из документации, hjson какбы умеет парсить "кривой" json, но парсит его в map-ку.
		// Интереснее на выходе получить структурку: то есть мы вначале конфиг преобразуем в map-ку, затем эту map-ку
		// сериализуем в json, а потом json превращаем в структурку. Не очень эффективно, но он и нечасто требуется.
		var (
			sampleBlacklist MyBlackList
			tmp             map[string]interface{}
		)

		err = hjson.Unmarshal(buf, &tmp)

		// Не удалось распарсить - попробуем следующего кандидата
		if err != nil {
			log.Warnf("Skip parsing blacklist file %s: %s", location, err)

			continue
		}

		tmpJSON, err := json.Marshal(tmp)

		// Не удалось преобразовать map-ку в json
		if err != nil {
			log.Warnf("Skip parsing blacklist file %s: %s", location, err)

			continue
		}

		if err := json.Unmarshal(tmpJSON, &sampleBlacklist); err != nil {
			log.Warnf("Skip parsing whitelist file %s: %s", location, err)

			continue
		}

		j.BlackList = sampleBlacklist
		blacklistLoaded = true

		log.Infof("Using %s as blacklist file", location)

		break
	}

	if !blacklistLoaded {
		return errors.New("blacklist was not loaded") //nolint:goerr113
	}

	return err
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
