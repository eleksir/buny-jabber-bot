package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jbrukh/bayesian"
	log "github.com/sirupsen/logrus"
)

const (
	goodDictionaryFile string = "./data/good_dictionary.txt"
	badDictionaryFile  string = "./data/bad_dictionary.txt"
	dataFile           string = "./data/data.bin"
)

// Выдаёт "очки похожести" для данной фразы.
func checkPhrase(s string) error {
	classifier, err := bayesian.NewClassifierFromFile(dataFile)

	if err != nil {
		return fmt.Errorf("unable to open %s: %w", dataFile, err)
	}

	s = nStringLower(s)

	scores, _, _ := classifier.LogScores(strings.Split(s, " "))

	_, err = fmt.Printf("Score %v\n", scores[0])

	return err
}

// Выучивает слова из предопределённых словарей.
func learn() error {
	classifier := bayesian.NewClassifier(Bad, Good)

	if err := feed(classifier, goodDictionaryFile, Good); err != nil {
		return err
	}

	if err := feed(classifier, badDictionaryFile, Bad); err != nil {
		return err
	}

	if err := classifier.WriteToFile(dataFile); err != nil {
		return fmt.Errorf("unable to save data to file %s: %w", dataFile, err)
	}

	return nil
}

// Дополняет заданный класс фразами из указанного файла.
func feed(c *bayesian.Classifier, filename string, bc bayesian.Class) error {
	fh, err := os.Open(filename)

	if err != nil {
		return fmt.Errorf("unable to open file %s: %w", filename, err)
	}

	defer func(fh *os.File) {
		if err := fh.Close(); err != nil {
			log.Errorf("unable to close %s cleanly: %s", filename, err)
		}
	}(fh)

	reader := bufio.NewReader(fh)

	for {
		line, err := reader.ReadString('\n')

		line = nStringLower(line)

		if err != nil {
			// Правильно обрабатываем последнюю строку и конец файла.
			if err == io.EOF {
				if line != "" {
					stuff := strings.Split(line, " ")
					c.Learn(stuff, bc)
				}

				break
			}

			return fmt.Errorf("unable to read %s: %w", filename, err)
		}

		if line != "" {
			stuff := strings.Split(line, " ")
			c.Learn(stuff, bc)
		}
	}

	return nil
}
