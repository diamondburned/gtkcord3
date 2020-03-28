#include <glib-2.0/glib.h>

gchar* error_message(GError *err) {
	return err->message;
}
