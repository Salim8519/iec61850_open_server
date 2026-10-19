"""Hand-written malformed (and a few valid) XML inputs covering every expat/defusedxml error class."""

NS = 'xmlns="http://www.iec.ch/61850/2003/SCL"'
DECL = '<?xml version="1.0" encoding="UTF-8"?>'
WIDE = chr(0xE9) + chr(0x20AC) + chr(0x1F600)

# Complete documents, written as text (encoded UTF-8 unless they are bytes already).
STANDALONE = [
    "", " ", "\n\n", "<", "<>", "< SCL/>", "<SCL", "<SCL ", "<SCL a>", "<SCL a=1/>", "<SCL a='1\"/>",
    "<SCL/><", "<SCL/>x", "<SCL/><SCL/>", "<SCL></SCL", "<SCL></>", "</SCL>", "<SCL>", "<SCL><IED>",
    "<SCL><IED></SCL>", "<SCL></IED>", "<SCL><a></b></SCL>", '<SCL a="1" a="2"/>', '<SCL a="1" b="2" a="3"/>',
    "<SCL>&</SCL>", "<SCL>&foo;</SCL>", "<SCL>&amp</SCL>", "<SCL a='<'/>", "<SCL a='&'/>", "<1SCL/>",
    "<-SCL/>", "<SCL\x01/>", "<SCL>\x01</SCL>", "<SCL>\x0b</SCL>", "<SCL><!-- x", "<SCL><!-- a -- b --></SCL>",
    "<SCL><!-- x --->", "<SCL><?pi", "<SCL><?xml version='1.0'?></SCL>", " <?xml version='1.0'?><SCL/>",
    "\n" + DECL + "<SCL/>", "<?xml?><SCL/>", "<?xml version='1.0' encoding='UTF-8' ?x?><SCL/>",
    "<?xml encoding='UTF-8'?><SCL/>", "<?xml version='2.0'?><SCL/>", "<?xml version='1.0' standalone='maybe'?><SCL/>",
    "<?xml version='1.0' encoding='x-bogus'?><SCL/>", '<?xml version="1.0" encoding="UTF-16"?><SCL/>',
    "<?xml version='1.0' encoding=''?><SCL/>", "<?XML version='1.0'?><SCL/>", "<SCL>&#0;</SCL>",
    "<SCL>&#xD800;</SCL>", "<SCL>&#x110000;</SCL>", "<SCL>&#99999999999;</SCL>", "<SCL>&#x;</SCL>",
    "<SCL>&#65</SCL>", "<SCL>&#xFFFF;</SCL>", "<SCL><![CDATA[x</SCL>", "<SCL><![CDATA[x]]</SCL>",
    "<SCL><![CDATA [x]]></SCL>", "<SCL>]]></SCL>", "<a:SCL/>", '<SCL b:x="1"/>', '<SCL xmlns:xml="urn:x"/>',
    '<SCL xmlns:xmlns="urn:x"/>', '<SCL xmlns:a=""/>', '<SCL xmlns:x="http://www.w3.org/XML/1998/namespace"/>',
    '<SCL xmlns:x="http://www.w3.org/2000/xmlns/"/>', '<xmlns:SCL/>', '<SCL xmlns="http://www.w3.org/2000/xmlns/"/>',
    '<SCL a:b="1" xmlns:a="urn:a" c:b="2" xmlns:c="urn:a"/>', "<SCL:a/>", "<SCL><IED name='A'/><IED name='A'/></SCL>",
    "<SCL><IED name='1A'/></SCL>", "<SCL><IED/></SCL>", "<SCL><IED name=''/></SCL>", "<SCL><IED name='it&apos;s'/></SCL>",
    "<SCL><IED name='a b'/></SCL>", f"<SCL><IED name='{chr(0xC4)}1'/></SCL>", "<SCL><IED name='A'/><IED name='B'/></SCL>",
    "<scl/>", "<Root/>", '<x:SCL xmlns:x="urn:other"/>', f"<SCL {NS}/>", "<SCL><x:IED xmlns:x='urn:o' name='9'/></SCL>",
    "<!DOCTYPE SCL><SCL/>", "<!DOCTYPE SCL []><SCL/>", '<!DOCTYPE SCL [<!ENTITY x "y">]><SCL/>',
    '<!DOCTYPE SCL [<!ENTITY x "y">]><SCL>&x;</SCL>', '<!DOCTYPE SCL [<!ENTITY x SYSTEM "file:///etc/passwd">]><SCL>&x;</SCL>',
    '<!DOCTYPE SCL [<!ENTITY % p "x">]><SCL/>', '<!DOCTYPE SCL [<!ENTITY u SYSTEM "u.bin" NDATA png>]><SCL/>',
    '<!DOCTYPE SCL SYSTEM "scl.dtd"><SCL/>', '<!DOCTYPE SCL PUBLIC "-//X//EN" "x.dtd"><SCL/>',
    '<!DOCTYPE SCL [<!ELEMENT SCL ANY>]><SCL/>', '<!DOCTYPE SCL [<!ATTLIST SCL a CDATA "d">]><SCL/>',
    "<!DOCTYPE SCL [", "<!DOCTYPE SCL><!DOCTYPE SCL><SCL/>", "<SCL/><!DOCTYPE SCL>", "<!DOCTYPE>",
    "<SCL>\n<IED>\n</SCL>\n", "<SCL>\n\n\n   <a></b>\n</SCL>", "<SCL>\r\n<a>\r\n</b>\r\n</SCL>", "<SCL>\r<a>\r</b></SCL>",
    "\t<SCL>\n\t\t&bad;\n</SCL>", f"<SCL desc='{WIDE}'><a></b></SCL>", f"<SCL>{WIDE}{WIDE}&x;</SCL>",
    f"<SCL>\n{WIDE}\n{WIDE}<\n</SCL>", f"<{chr(0xC9)}l{chr(0xE9)}ment/>", "<SCL/>\x00", "\x00<SCL/>",
    "\ufeff<SCL/>", "\ufeff\ufeff<SCL/>", "<SCL>\ufffe</SCL>", "<SCL a='\ufffe'/>",
]

STANDALONE_BYTES = [
    b"\xef\xbb\xbf", b"\xef\xbb", b"\xef\xbb\xbf<SCL/>", b"\xef\xbb\xbf\xef\xbb\xbf<SCL/>", b"<SCL>\xff</SCL>",
    b"<SCL>\xc3</SCL>", b"<SCL>\xc3", b"<SCL>\xe2\x82", b"<SCL>\xe2\x82\xac</SCL>", b"<SCL>\xf0\x9f\x98</SCL>",
    b"<SCL>\xc0\xaf</SCL>", b"<SCL>\xed\xa0\x80</SCL>", b"<SCL>\xf4\x90\x80\x80</SCL>", b"<SCL>\x80</SCL>",
    b"<SCL a='\xff'/>", b"<SCL\xc3\xa9/>", b"<\xc3\xa9/>", b"\xff\xfe<\x00S\x00C\x00L\x00/\x00>\x00",
    b"\xfe\xff\x00<\x00S\x00C\x00L\x00/\x00>", b"\xff\xfe<\x00S\x00C\x00L\x00", b"\xff\xfe<\x00S\x00C\x00L\x00/\x00>",
    b"\xff\xfe<\x00", b"\xff\xfe<\x00S\x00C\x00L\x00/\x00>\x00\x00", b"<\x00S\x00C\x00L\x00/\x00>\x00",
    b"\x00<\x00S\x00C\x00L\x00/\x00>", b"\xff\xfe\x00\x00<\x00\x00\x00", b"\x00\x00\xfe\xff",
    b'<?xml version="1.0" encoding="ISO-8859-1"?><SCL desc="\xe9\x81\xff"/>',
    b'<?xml version="1.0" encoding="windows-1252"?><SCL desc="\x80\x81"/>',
    b'<?xml version="1.0" encoding="windows-1252"?><SCL desc="\x9f"/>',
    b'<?xml version="1.0" encoding="cp1252"?><SCL/>', b'<?xml version="1.0" encoding="latin1"?><SCL/>',
    b'<?xml version="1.0" encoding="US-ASCII"?><SCL desc="\xe9"/>', b'<?xml version="1.0" encoding="ascii"?><SCL/>',
    b'<?xml version="1.0" encoding="utf8"?><SCL/>', b'<?xml version="1.0" encoding="UTF-32"?><SCL/>',
    b'<?xml version="1.0" encoding="Shift_JIS"?><SCL desc="\x82\xa0"/>', b'<?xml version="1.0" encoding="koi8-r"?><SCL/>',
    b'<?xml version="1.0" encoding="iso-8859-15"?><SCL desc="\xa4"/>', b'<?xml version="1.0" encoding="big5"?><SCL/>',
    b'\xef\xbb\xbf<?xml version="1.0" encoding="ISO-8859-1"?><SCL/>', b'\xef\xbb\xbf<?xml version="1.0" encoding="UTF-16"?><SCL/>',
    b'\xff\xfe' + '<?xml version="1.0" encoding="ISO-8859-1"?><SCL/>'.encode("utf-16-le"),
    b'\xfe\xff' + '<?xml version="1.0" encoding="UTF-16LE"?><SCL/>'.encode("utf-16-be"),
    b'\xff\xfe' + '<?xml version="1.0" encoding="UTF-16LE"?><SCL/>'.encode("utf-16-le"),
    '<?xml version="1.0" encoding="UTF-16LE"?><SCL/>'.encode("utf-16-le"),
    '<?xml version="1.0" encoding="UTF-16BE"?><SCL/>'.encode("utf-16-be"),
]

# Fragments placed inside the root element in several contexts.
BODY = [
    "&foo;", "&", "&#0;", "&#x1;", "&#xD800;", "<a></b>", "<a>", "</a>", "<!-- x", "<!-- a -- b -->", "<![CDATA[x",
    "<a b='1' b='2'/>", "<p:x/>", "<x p:a='1'/>", "\x01", "\x1f", "<?xml version='1.0'?>", "<?pi", "</SCL>junk",
    "</SCL><SCL/>", "<a b=c/>", "<a b='c/>", "<a\tb='1'c='2'/>", "<1a/>", "<a/ >", "< a/>", "<a>]]></a>",
    "<!DOCTYPE x>", "<![CDATA[ok]]><!", "<a xmlns:p='urn:p'><p:b></b></a>", "<a xmlns:xml='urn:x'/>",
    "<IED name='A'/><IED name='A'/>", "<IED name='9A'/>", "<IED/>", "<a b='&#60;'/><c d='<'/>", "&#x110000;",
    "<a>&lt;&gt;&amp;&quot;&apos;</a>&nope;", "<a b='\ufffe'/>", "<a>text\x0c</a>", "<a/><b></c>",
]


def contexts(fragment):
    """The same fragment inside a single-line, multi-line, CRLF, non-ASCII, UTF-16 and Latin-1 document."""
    multi = f"{DECL}\n<SCL {NS} version=\"2007\">\n  <Header id=\"h\"/>\n  <IED name=\"A\"/>\n  {fragment}\n</SCL>\n"
    wide = f"<SCL desc=\"{WIDE}\">{WIDE}\n{WIDE}{fragment}</SCL>"
    utf16 = '<?xml version="1.0" encoding="UTF-16"?><SCL>' + fragment + "</SCL>"
    latin = '<?xml version="1.0" encoding="ISO-8859-1"?>\n<SCL desc="\xe9">\xe9' + fragment + "</SCL>"
    return [("<SCL>" + fragment + "</SCL>").encode("utf-8"), multi.encode("utf-8"),
            multi.replace("\n", "\r\n").encode("utf-8"), wide.encode("utf-8"),
            b"\xff\xfe" + utf16.encode("utf-16-le", "surrogatepass"), latin.encode("latin-1", "replace")]


def snippets():
    result = [text.encode("utf-8", "surrogatepass") for text in STANDALONE] + list(STANDALONE_BYTES)
    for fragment in BODY:
        result.extend(contexts(fragment))
    return result
