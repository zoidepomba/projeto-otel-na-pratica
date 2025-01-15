package log

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
    InfoLogger *log.Logger
    ErrorLogger *log.Logger
}

//NewLogger cria uma nova instancia do lo
func NewLogeer () *Logger {
    return &Logger {
        InfoLogger: log.New(os.Stdout, "INFO: ", log.Ldate | log.Ltime ),
        ErrorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate | log.Ltime ),
    }

}


func (l *Logger) Info(message string, args ...interface{}) {
    formattedMessage := fmt.Sprintf(message, args...)
    l.InfoLogger.Println(formattedMessage)
}

// Error registra uma mensagem de erro com múltiplos argumentos
func (l *Logger) Error(message string, args ...interface{}) {
    formattedMessage := fmt.Sprintf(message, args...)
    l.ErrorLogger.Println(formattedMessage)
}