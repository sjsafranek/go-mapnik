package mapnik

// #cgo CXXFLAGS: -I/usr/include -I/usr/include/mapnik/agg -I/usr/include -I/usr/include/freetype2 -I/usr/include/libxml2 -I/usr/include/gdal -I/usr/include/postgresql -I/usr/include/python2.7 -I/usr/include/python2.7 -I/usr/include/cairo -I/usr/include/glib-2.0 -I/usr/lib/x86_64-linux-gnu/glib-2.0/include -I/usr/include/pixman-1 -I/usr/include/libpng12 -D_FORTIFY_SOURCE=2 -g0 -DHAVE_JPEG -DMAPNIK_USE_PROJ4 -DHAVE_PNG -DHAVE_TIFF -DBIGINT -DBOOST_REGEX_HAS_ICU -DLINUX -DMAPNIK_THREADSAFE -DBOOST_SPIRIT_NO_PREDEFINED_TERMINALS=1 -DBOOST_PHOENIX_NO_PREDEFINED_TERMINALS=1 -DNDEBUG -DHAVE_CAIRO -DHAVE_LIBXML2 -g -O2 -fstack-protector-strong -Wformat -Werror=format-security -g0 -ansi -Wall -pthread -O2 -fno-strict-aliasing -finline-functions -Wno-inline -Wno-parentheses -Wno-char-subscripts
// #cgo LDFLAGS: -L/usr/lib -lmapnik -lboost_system
import "C"

const (
	fontPath   = "/usr/share/fonts"
	pluginPath = "/usr/lib/mapnik/2.2/input"
)
