package src

import (
	"bufio"
	"os"
)

func ReadLine(filename string) ([]string, error) {
	peers := []string{}
	file, err := os.Open(filename)
    if err != nil {
        return peers, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        peers = append(peers, line)
    }
    if err := scanner.Err(); err != nil {
        return peers, err
    }
    return peers, nil
}

func CreateWriteLine(filename, addrName string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    if _, err := file.WriteString(addrName + "\n"); err != nil {
        return err
    }
    return nil
}

func WriteLine(filename, addrName string) error {
    file, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return err
    }
    defer file.Close()

    if _, err := file.WriteString(addrName + "\n"); err != nil {
        return err
    }
    return nil
}
