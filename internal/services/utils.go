package services

import (
    "os"
    "io"
    "bufio"
    "time"
    "log/slog"
    "fmt"
    "errors"
)

const (
    TSFilePath string = "nats_app/created/.created"
)

// build control file at first start
func MakeNewTSFile(log *slog.Logger) error {
    file, err := os.Create(TSFilePath)
    if err != nil {
        log.Error(fmt.Sprintf("Path %s not exists.", TSFilePath))
        return errors.New("<.create> path not exists...")
    }
    defer file.Close()
    ts := time.Now()
    writer := bufio.NewWriter(file)
    bytes, err := ts.MarshalBinary()
    if err != nil {
        log.Error("Can`t Marshal ts to bytes...")
        return errors.New("Error in TS Marshal")
    }
    writer.Write(bytes)
    return nil
}

func UpdateTimestamp(log *slog.Logger) error {
    file, err := os.OpenFile(TSFilePath, os.O_RDWR, 0666)
    if err != nil {
        log.Error(fmt.Sprintf("Path %s not exists.", TSFilePath))
        return errors.New("Can`t open <.create> file...")
    }
    defer file.Close()
    ts := time.Now()
    writer := bufio.NewWriter(file)
    bytes, err := ts.MarshalBinary()
    if err != nil {
        log.Error("Can`t Marshal ts to bytes...")
        return errors.New("Error in TS Marshal")
    }
    writer.Write(bytes)
    return nil
}

func GetPreviousTS(log *slog.Logger) (time.Time, error) {
    file, err := os.OpenFile(TSFilePath, os.O_RDWR, 0666)
    if err != nil {
        log.Error(fmt.Sprintf("Path %s not exists.", TSFilePath))
        return time.Now(), errors.New("Can`t open <.create> ffile...")
    }
    defer file.Close()
    buff := make([]byte, 32)
    reader := bufio.NewReader(file)
    ts := time.Now()
    for {
        _, err := reader.Read(buff)
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Error("Error on <.create> reading...")
            return time.Now(), errors.New("Read error...")
        }
    }
    err = ts.UnmarshalBinary(buff)
    if err != nil {
        log.Error(fmt.Sprintf("Inalid buf: %+v, %v", buff, ts))
        return time.Now(), errors.New("Can`t Unmarshal TS...")
    }
    return ts, nil
}
