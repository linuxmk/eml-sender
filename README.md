This is a console based application written in Go lang.
The main puropse of this application is to test if a given mail header is a valid one, and
extract the two data from it. Those two piece of information are: display data and email address.

You can run it from a console. 
If you type the name of the application without any paramtets it will show you how you can use it.

$go run eml-sender.go
Usage: eml-sender file.eml
Usage: eml-sender tests

you can run and validate an .eml file or you can run tests.

This is simple implementation, that does not gurantee that will work 100%, with all possible way to detect display name and email.
In this small project for detection am using regex, but for more accurate implementation it suggest a state parser following 
the rfc 5322 rules.

