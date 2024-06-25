/*
- @Author: aztec
- @Date: 2024-06-14 16:30:45
- @Description:
- @
- @Copyright (c) 2024 by aztec, All Rights Reserved.
*/
package compatiblelogger

import "fmt"

type Logger struct {
	logPrefix  string
	fnLogDebug func(string)
	fnLogError func(string)
}

func (l *Logger) InitCompatibleLogger(fnLogDebug, fnLogError func(string), logPrefix string) {
	l.fnLogDebug = fnLogDebug
	l.fnLogError = fnLogError
	l.logPrefix = logPrefix
}

func (l Logger) LogDebug(format string, params ...interface{}) {
	if l.fnLogDebug == nil {
		l.fnLogDebug = func(s string) {
			fmt.Println(s)
		}
	}

	if len(l.logPrefix) > 0 {
		l.fnLogDebug(fmt.Sprintf(l.logPrefix+" "+format, params...))
	} else {
		l.fnLogDebug(fmt.Sprintf(format, params...))
	}
}

func (l Logger) LogError(format string, params ...interface{}) {
	if l.fnLogError == nil {
		l.fnLogError = func(s string) {
			fmt.Println(s)
		}
	}

	if len(l.logPrefix) == 0 {
		l.fnLogError(fmt.Sprintf(l.logPrefix+" "+format, params...))
	} else {
		l.fnLogError(fmt.Sprintf(format, params...))
	}
}
