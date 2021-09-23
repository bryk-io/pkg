package log

import "sync"

type composite struct {
	mu   sync.Mutex
	list []Logger
}

// Composite allows to combine an control multiple logger instances
// through a single common interface. This is useful, for example, when
// you wanna save structured logs to a file while at same time displaying
// textual messages to standard output and/or sending messages to some
// aggregation system.
func Composite(ll ...Logger) Logger {
	return &composite{
		list: ll,
	}
}

func (c *composite) Debug(args ...interface{}) {
	for _, el := range c.list {
		el.Debug(args...)
	}
}

func (c *composite) Debugf(format string, args ...interface{}) {
	for _, el := range c.list {
		el.Debugf(format, args...)
	}
}

func (c *composite) Info(args ...interface{}) {
	for _, el := range c.list {
		el.Info(args...)
	}
}

func (c *composite) Infof(format string, args ...interface{}) {
	for _, el := range c.list {
		el.Infof(format, args...)
	}
}

func (c *composite) Warning(args ...interface{}) {
	for _, el := range c.list {
		el.Warning(args...)
	}
}

func (c *composite) Warningf(format string, args ...interface{}) {
	for _, el := range c.list {
		el.Warningf(format, args...)
	}
}

func (c *composite) Error(args ...interface{}) {
	for _, el := range c.list {
		el.Error(args...)
	}
}

func (c *composite) Errorf(format string, args ...interface{}) {
	for _, el := range c.list {
		el.Errorf(format, args...)
	}
}

func (c *composite) Panic(args ...interface{}) {
	for _, el := range c.list {
		el.Panic(args...)
	}
}

func (c *composite) Panicf(format string, args ...interface{}) {
	for _, el := range c.list {
		el.Panicf(format, args...)
	}
}

func (c *composite) Fatal(args ...interface{}) {
	for _, el := range c.list {
		el.Fatal(args...)
	}
}

func (c *composite) Fatalf(format string, args ...interface{}) {
	for _, el := range c.list {
		el.Fatalf(format, args...)
	}
}

func (c *composite) Print(level Level, args ...interface{}) {
	for _, el := range c.list {
		el.Print(level, args...)
	}
}

func (c *composite) Printf(level Level, format string, args ...interface{}) {
	for _, el := range c.list {
		el.Printf(level, format, args...)
	}
}

func (c *composite) WithFields(fields Fields) Logger {
	c.mu.Lock()
	for i, el := range c.list {
		c.list[i] = el.WithFields(fields)
	}
	c.mu.Unlock()
	return c
}

func (c *composite) WithField(key string, value interface{}) Logger {
	c.mu.Lock()
	for i, el := range c.list {
		c.list[i] = el.WithField(key, value)
	}
	c.mu.Unlock()
	return c
}

func (c *composite) Sub(tags Fields) Logger {
	cs := &composite{
		list: make([]Logger, len(c.list)),
	}
	c.mu.Lock()
	for i, el := range c.list {
		cs.list[i] = el.Sub(tags)
	}
	c.mu.Unlock()
	return cs
}
