# Copyright (c) 2015-2016 Stefan Tatschner
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

MAN_MDS  = $(wildcard *.1.md) $(wildcard *.7.md)
DOC_MAN  = $(patsubst %.md,%,$(MAN_MDS))
DOC_HTML = $(patsubst %.md,%.html,$(MAN_MDS))
INCLUDES = footer.md

all: html man

html: $(DOC_HTML)

man: $(DOC_MAN)

clean:
	$(RM) *.1 *.7 *.html

% : %.md $(INCLUDES)
	cat $< $(INCLUDES) > /tmp/concat.md
	pandoc --standalone --to man -o $@ /tmp/concat.md
	$(RM) /tmp/concat.md

%.html : %.md $(INCLUDES)
	cat $< $(INCLUDES) > /tmp/concat.md
	pandoc --standalone --to html -o $@ /tmp/concat.md
	$(RM) /tmp/concat.md

.PHONY: clean man html
