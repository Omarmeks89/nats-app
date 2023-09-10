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
    TSFilePath string = "app.created"
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
    log.Error(fmt.Sprintf("%v created bytes", bytes))
    if err != nil {
        log.Error("Can`t Marshal ts to bytes...")
        return errors.New("Error in TS Marshal")
    }
    writer.Write(bytes)
    writer.Flush()
    return nil
}

func UpdateTimestamp(t time.Time, log *slog.Logger) error {
    file, err := os.OpenFile(TSFilePath, os.O_RDWR, 0666)
    if err != nil {
        log.Error(fmt.Sprintf("Path %s not exists.", TSFilePath))
        return errors.New("Can`t open <.create> file...")
    }
    defer file.Close()
    writer := bufio.NewWriter(file)
    bytes, err := t.MarshalBinary()
    if err != nil {
        log.Error("Can`t Marshal ts to bytes...")
        return errors.New("Error in TS Marshal")
    }
    writer.Write(bytes)
    writer.Flush()
    return nil
}

func GetPreviousTS(log *slog.Logger) (time.Time, error) {
    log.Debug("GetPreviousTS")
    file, err := os.OpenFile(TSFilePath, os.O_RDWR, 0666)
    if err != nil {
        log.Error(fmt.Sprintf("Path %s not exists.", TSFilePath))
        return time.Now(), errors.New("Can`t open <.create> file...")
    }
    defer file.Close()
    buff := make([]byte, 15)
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
        log.Error(fmt.Sprintf("Inalid buf: %+v, %v, %s", buff, ts, err.Error()))
        return time.Now(), errors.New("Can`t Unmarshal TS...")
    }
    return ts, nil
}

func CheckCrashed() bool {
    if _, err := os.Stat(TSFilePath); err == nil {
        // path exists, means we have running earlier
        return true
    }
    return false
}
