// syslog is a simple io.Writer that writes to syslog over a network connection.
// The difference with the buildin syslog package
// is that it will detect the priority based on [ERR] [WARN] or [INFO] tags.
//
// The main usage of this package is to import it for the side effects
// cause by setting the SYSLOG environment variable. Setting this variable will
// change the default writer of the log package.
//
// SYSLOG should be set to network:address, for example udp:localhost:514
package syslog
