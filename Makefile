include $(GOROOT)/src/Make.inc

TARG=metrics
GOFILES=\
	edsample.go\
	ewma.go\
	histogram.go\
	uniformsample.go\
	util.go\

include $(GOROOT)/src/Make.pkg