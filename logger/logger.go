package logger

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

const calldepth = 3

var (
	Silent           bool
	Verbose          bool
	Color            bool
	stdOutLogger     = log.New(os.Stdout, "", 0)
	stdOutWarnLogger = log.New(os.Stdout, "WARNING: ", 0)
	stdErrLogger     = log.New(os.Stderr, "ERROR: ", 0)
)

func StdErrOutput(b []byte) (n int, err error) {
	if Color {
		b = append([]byte(ColorRed), b...)
		b = append(b, ColorNC...)
	}
	return os.Stderr.Write([]byte(b))
}

// io.Writer interface implementation. To use with error logs, pass StdErrOutput func:
// Writer(StdErrOutput) <- implements io.Writer
type Writer func(b []byte) (n int, err error)

func (w Writer) Write(b []byte) (n int, err error) {
	return w(b)
}

func Error(v ...interface{}) {
	Log(stdErrLogger, ColorRed, v...)
}

func Errorf(format string, v ...interface{}) {
	Logf(stdErrLogger, ColorRed, format, v...)
}

func Warn(v ...interface{}) {
	Log(stdOutWarnLogger, ColorLightRed, v...)
}

func Warnf(format string, v ...interface{}) {
	Logf(stdOutWarnLogger, ColorLightRed, format, v...)
}

func Heading(v ...interface{}) {
	if !Silent {
		Log(stdOutLogger, ColorGreen, v...)
	}
}

func Headingf(format string, v ...interface{}) {
	if !Silent {
		Logf(stdOutLogger, ColorGreen, format, v...)
	}
}

func Info(v ...interface{}) {
	if !Silent {
		Log(stdOutLogger, ColorCyan, v...)
	}
}

func Infof(format string, v ...interface{}) {
	if !Silent {
		Logf(stdOutLogger, ColorCyan, format, v...)
	}
}

func Debug(v ...interface{}) {
	if Verbose && !Silent {
		Log(stdOutLogger, ColorLightGrey, v...)
	}
}

func Debugf(format string, v ...interface{}) {
	if Verbose && !Silent {
		Logf(stdOutLogger, ColorLightGrey, format, v...)
	}
}

func Log(l *log.Logger, color string, v ...interface{}) {
	msg := fmt.Sprint(v...)
	if Color {
		msg = colorizeMessage(color, msg)
	}
	l.Output(calldepth, msg)
}

func Logf(l *log.Logger, color, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if Color {
		msg = colorizeMessage(color, msg)
	}
	l.Output(calldepth, msg)
}

func colorizeMessage(color, s string) string {
	whitespace := regexp.MustCompile(`\s*$`)
	trimmed := whitespace.ReplaceAllString(s, "")
	trailing := whitespace.FindString(s)

	return color + trimmed + ColorNC + trailing
}
