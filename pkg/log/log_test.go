package log

import (
	"testing"
)

func TestPrintf(t *testing.T) {
	log := New("TESTER TOO LONG")
	log.Printf("printf message")
	log.Warnf("warnf message")
	log.Errorf("warnf message")
	log.Debugf("debugf message")
	log.Highlightf("highlight message")
	log.Eventf("eventf message")
	log.Infof("infof message")

	log.SetPrinter(ColorPrinter)
	log.Printf("printf message")
	log.Warnf("warnf message")
	log.Errorf("warnf message")
	log.Debugf("debugf message")
	log.Highlightf("highlight message")
	log.Eventf("eventf message")
	log.Infof("infof message")

	log.SetPrinter(BasicPrinter)
	log.Printf("printf message")
	log.Warnf("warnf message")
	log.Errorf("warnf message")
	log.Debugf("debugf message")
	log.Highlightf("highlight message")
	log.Eventf("eventf message")
	log.Infof("infof message")

}
