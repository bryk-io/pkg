package log

import (
	stdL "log"
	"os"
	"testing"
)

func TestWithStandard(t *testing.T) {
	ll := stdL.New(os.Stdout, "", stdL.LstdFlags)
	log := WithStandard(ll)

	sampleFields := Fields{
		"foo": 1,
		"bar": true,
		"baz": "application",
	}

	t.Run("Sub", func(t *testing.T) {
		sub := log.Sub(Fields{"prefix": "sub"})
		sub.Debug("testing a debug message")
		sub.WithFields(sampleFields).Debug("this message has fields")
		sub.Info("testing a debug message")
		sub.WithFields(sampleFields).Info("this message has fields")
		sub.Warning("testing a debug message")
		sub.WithFields(sampleFields).Warning("this message has fields")
		sub.Error("testing a debug message")
		sub.WithFields(sampleFields).Error("this message has fields")
	})

	t.Run("WithField", func(t *testing.T) {
		log.WithField("dimension", "c137").Debug("single field")
		log.WithField("dimension", "c137").Info("single field")
		log.WithField("dimension", "c137").Warning("single field")
		log.WithField("dimension", "c137").Error("single field")
	})

	t.Run("Debug", func(t *testing.T) {
		log.Debug("testing a debug message")
		log.WithFields(sampleFields).Debug("this message has fields")
		log.Debug("without fields")
		log.WithFields(sampleFields).Debug("final test")
		log.Debugf("formatted: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Debugf("formatted: %+v", Fields{"foo": "bar"})
		log.Print(Debug, "simple print")
		log.WithFields(sampleFields).Print(Debug, "print with fields")
		log.Printf(Debug, "formatted print: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Printf(Debug, "formatted print: %+v", Fields{"foo": "bar"})
	})

	t.Run("Info", func(t *testing.T) {
		log.Info("testing a debug message")
		log.WithFields(sampleFields).Info("this message has fields")
		log.Info("without fields")
		log.WithFields(sampleFields).Info("final test")
		log.Infof("formatted: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Infof("formatted: %+v", Fields{"foo": "bar"})
		log.Print(Info, "simple print")
		log.WithFields(sampleFields).Print(Info, "print with fields")
		log.Printf(Info, "formatted print: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Printf(Info, "formatted print: %+v", Fields{"foo": "bar"})
	})

	t.Run("Warning", func(t *testing.T) {
		log.Warning("testing a debug message")
		log.WithFields(sampleFields).Warning("this message has fields")
		log.Warning("without fields")
		log.WithFields(sampleFields).Warning("final test")
		log.Warningf("formatted: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Warningf("formatted: %+v", Fields{"foo": "bar"})
		log.Print(Warning, "simple print")
		log.WithFields(sampleFields).Print(Warning, "print with fields")
		log.Printf(Warning, "formatted print: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Printf(Warning, "formatted print: %+v", Fields{"foo": "bar"})
	})

	t.Run("Error", func(t *testing.T) {
		log.Error("testing a debug message")
		log.WithFields(sampleFields).Error("this message has fields")
		log.Error("without fields")
		log.WithFields(sampleFields).Error("final test")
		log.Errorf("formatted: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Errorf("formatted: %+v", Fields{"foo": "bar"})
		log.Print(Error, "simple print")
		log.WithFields(sampleFields).Print(Error, "print with fields")
		log.Printf(Error, "formatted print: %+v", Fields{"foo": "bar"})
		log.WithFields(sampleFields).Printf(Error, "formatted print: %+v", Fields{"foo": "bar"})
	})

	t.Run("Panic", func(t *testing.T) {
		t.Run("Simple", func(t *testing.T) {
			defer func() {
				recover()
			}()
			log.Panic("testing a debug message")
		})

		t.Run("WithFields", func(t *testing.T) {
			defer func() {
				recover()
			}()
			log.WithFields(sampleFields).Panic("this message has fields")
		})
	})
}

func TestDiscard(t *testing.T) {
	log := Discard()
	log.Debug("none of this messages")
	log.Info("will produce any output")
	log.Warning("but the component respects")
	log.Error("the same API so is easy to")
	log.Panic("switch at runtime")
	log.Fatal("not even panics and crashes are executed for 'discard'")
}

func ExampleWithStandard() {
	// Setup a logger using the standard library package
	// Renamed to "stdL" in this example.
	ll := stdL.New(os.Stdout, "", 0)

	// Create logger instance
	log := WithStandard(ll)

	// Use log handler
	log.Debug("use log handler now")
	log.WithFields(Fields{
		"foo": 1,
		"bar": true,
		"baz": "application",
	}).Info("loggers support structured information")
}

func ExampleDiscard() {
	log := Discard()
	log.Debug("none of this messages")
	log.Info("will produce any output")
	log.Warning("but the component respects")
	log.Error("the same API so is easy to")
	log.Panic("switch at runtime")
	log.Fatal("not even panics and crashes are executed for 'discard'")
}
