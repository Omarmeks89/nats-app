package services

type DBConnectionLost struct {
    msg string
    err error
}

func (e *DBConnectionLost) Error() string {
    return (*e).msg
}

func (e *DBConnectionLost) Unwrap() error {
    return (*e).err
}

