package lcdm_2000_api_test

import (
	"fmt"
	"testing"
	"time"

	api "lcdm_2000_api"
)

func TestConnection(t *testing.T) {
	c, er := api.NewConnection("COM4", api.Baud9600, true, 3 * time.Second)

	if er != nil {
		fmt.Println(er)
		return
	}
//	er = c.Reset()
	s,e,r,r1,r2 := c.UpperDispense(1)

	fmt.Println(s,e,r,r1,r2)

	if er != nil {
		fmt.Println(er)
		return
	}
}
