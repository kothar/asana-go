package asana

import (
	"log"
)

func (c *Client) info(format string, args ...interface{}) {
	if len(c.Verbose) > 0 {
		log.Printf(format, args...)
	}
}

func (c *Client) trace(format string, args ...interface{}) {
	if len(c.Verbose) > 1 {
		log.Printf(format, args...)
	}
}

func (c *Client) debug(format string, args ...interface{}) {
	if c.Debug {
		log.Printf(format, args...)
	}
}

func (e *expandable) info(format string, args ...interface{}) {
	e.Client.info(format, args...)
}

func (e *expandable) trace(format string, args ...interface{}) {
	e.Client.trace(format, args...)
}

func (e *expandable) debug(format string, args ...interface{}) {
	e.Client.debug(format, args...)
}
