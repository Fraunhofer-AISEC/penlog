# SPDX-License-Identifier: Apache-2.0

import inspect
import json
import os
import socket
import sys
import traceback
import uuid
from datetime import datetime
from enum import Enum, IntEnum
from typing import Dict, List, TextIO, Optional


class MessageType(str, Enum):
    READ = "read"
    WRITE = "write"
    PREAMBLE = "preamble"
    MESSAGE = "message"
    SUMMARY = "summary"


class MessagePrio(IntEnum):
    EMERGENCY = 0
    ALERT = 1
    CRITICAL = 2
    ERROR = 3
    WARNING = 4
    NOTICE = 5
    INFO = 6
    DEBUG = 7


class OutputType(Enum):
    JSON = "json"
    JSON_PRETTY = "json-pretty"
    HR = "hr"
    HR_TINY = "hr-tiny"


class Color(Enum):
    NOP = ""
    RESET = "\033[0m"
    BOLD = "\033[1m"
    RED = "\033[31m"
    GREEN = "\033[32m"
    YELLOW = "\033[33m"
    BLUE = "\033[34m"
    PURPLE = "\033[35m"
    CYAN = "\033[36m"
    WHITE = "\033[37m"
    GRAY = "\033[0;38;5;245m"


def colorize(color: Color, s: str) -> str:
    if color == Color.NOP:
        return s
    return f"{color.value}{s}{Color.RESET.value}"


def _get_line_number(depth: int) -> str:
    stack = inspect.stack()
    frame = stack[depth]
    return f'{frame.filename}:{frame.lineno}'


def str2bool(s: str) -> bool:
    return s.lower() in ['true', '1', 't', 'y']


class HRFormatter:

    def __init__(self, show_colors: bool, show_ids: bool,
                 show_lines: bool, show_stacktraces: bool,
                 show_tags: bool, tiny: bool):
        self.show_colors = show_colors
        self.show_ids = show_ids
        self.show_lines = show_lines
        self.show_stacktraces = show_stacktraces
        self.show_tags = show_tags
        self.tiny = tiny

    @staticmethod
    def _colorize_data(data: str, prio: MessagePrio) -> str:
        if prio == MessagePrio.EMERGENCY or \
                prio == MessagePrio.ALERT or \
                prio == MessagePrio.CRITICAL or \
                prio == MessagePrio.ERROR:
            data = colorize(Color.BOLD, colorize(Color.RED, data))
        elif prio == MessagePrio.WARNING:
            data = colorize(Color.BOLD, colorize(Color.YELLOW, data))
        elif prio == MessagePrio.NOTICE:
            data = colorize(Color.BOLD, data)
        elif prio == MessagePrio.INFO:
            pass
        elif prio == MessagePrio.DEBUG:
            data = colorize(Color.GRAY, data)

    def format(self, msg: Dict) -> str:
        out = ""
        ts = datetime.fromisoformat(msg["timestamp"])
        ts_formatted = ts.strftime("%b %m %H:%M:%S.%f")[:-3]
        component = msg["component"]
        msgtype = msg["type"]
        data = msg["data"]
        if self.show_colors and "priority" in msg:
            prio = MessagePrio(msg["priority"])
            data = self._colorize_data(data, prio)
        if self.tiny:
            out = f"{ts_formatted}: {data}"
        else:
            out = f"{ts_formatted} {{{component: <8}}} [{msgtype: <8}]: {data}"
        if self.show_ids and "id" in msg:
            out += "\n"
            if self.show_colors:
                out += f" => id  : {colorize(Color.YELLOW, msg['id'])}"
            else:
                out += f" => id  : {msg['id']}"
        if self.show_lines and "line" in msg:
            out += "\n"
            if self.show_colors:
                out += f" => line: {colorize(Color.BLUE, msg['line'])}"
            else:
                out += f" => line: {msg['line']}"
        if self.show_tags and "tags" in msg:
            out += "\n"
            out += f" => tags: {' '.join(msg['tags'])}"
        if self.show_stacktraces and "stacktrace" in msg:
            out += "\n"
            out += " => stacktrace:\n"
            for line in msg['stacktrace'].splitlines():
                if self.show_colors:
                    out += colorize(Color.GRAY, f" | {line}\n")
                else:
                    out += f" | {line}\n"
        return out


class Logger:
    def __init__(self, component: str = "root", flush: bool = False,
                 file_: TextIO = sys.stderr,
                 loglevel: MessagePrio = MessagePrio.DEBUG,
                 output_type: Optional[OutputType] = None,
                 show_colors: bool = False):
        self.host = socket.gethostname()
        self.component = component
        self.flush = flush
        self.file = file_
        self.loglevel = loglevel
        if output_type:
            self.output_type = output_type
        else:
            output_type_raw = os.environ.get("PENLOG_OUTPUT")
            if output_type_raw is None:
                self.output_type = OutputType.HR_TINY
            else:
                self.output_type = OutputType(output_type_raw)
        self.lines = str2bool(os.environ.get("PENLOG_CAPTURE_LINES", ""))
        self.stacktraces = str2bool(os.environ.get("PENLOG_CAPTURE_STACKTRACES", ""))
        is_tiny = True if self.output_type == OutputType.HR_TINY else False
        self.hr_formatter = HRFormatter(show_colors, False, self.lines,
                                        self.stacktraces, False, is_tiny)

    def _log(self, msg: Dict, depth: int) -> None:
        if "priority" in msg:
            try:
                prio = MessagePrio(msg["priority"])
                if prio > self.loglevel:
                    return
            except ValueError:
                pass
        msg["id"] = str(uuid.uuid4())
        msg["component"] = self.component
        msg["host"] = self.host
        now = datetime.now().astimezone()
        msg["timestamp"] = now.isoformat()
        if self.lines:
            msg["line"] = _get_line_number(depth)
        if self.stacktraces:
            msg["stacktrace"] = ''.join(traceback.format_stack())
        if self.output_type == OutputType.JSON:
            print(json.dumps(msg), file=self.file, flush=self.flush)
        elif self.output_type == OutputType.JSON_PRETTY:
            print(json.dumps(msg, indent=2), file=self.file, flush=self.flush)
        elif self.output_type == OutputType.HR:
            out = self.hr_formatter.format(msg)
            print(out, file=self.file, flush=self.flush)
        elif self.output_type == OutputType.HR_TINY:
            out = self.hr_formatter.format(msg)
            print(out, file=self.file, flush=self.flush)
        else:
            print("invalid penlog output", file=sys.stderr)
            sys.exit(1)

    def _log_msg(self, data: str, type_: MessageType = MessageType.MESSAGE,
                 prio: MessagePrio = MessagePrio.INFO,
                 tags: Optional[List[str]] = None) -> None:
        msg = {
            'type': type_,
            'priority': prio,
            'data': data,
        }
        if tags:
            msg['tags'] = tags
        self._log(msg, 4)

    def log_msg(self, data: str, type_: MessageType = MessageType.MESSAGE,
                prio: MessagePrio = MessagePrio.INFO,
                tags: Optional[List[str]] = None) -> None:
        msg = {
            'type': type_,
            'priority': prio,
            'data': data,
        }
        if tags:
            msg['tags'] = tags
        self._log(msg, 3)

    def log_preamble(self, data: str) -> None:
        self._log_msg(data, MessageType.PREAMBLE, MessagePrio.NOTICE)

    def log_read(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.READ, MessagePrio.DEBUG, tags)

    def log_write(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.WRITE, MessagePrio.DEBUG, tags)

    def log_debug(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.MESSAGE, MessagePrio.DEBUG, tags)

    def log_info(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.MESSAGE, MessagePrio.INFO, tags)

    def log_notice(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.MESSAGE, MessagePrio.NOTICE, tags)

    def log_warning(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.MESSAGE, MessagePrio.WARNING, tags)

    def log_error(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.MESSAGE, MessagePrio.ERROR, tags)

    def log_critical(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.MESSAGE, MessagePrio.CRITICAL, tags)

    def log_summary(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.SUMMARY, MessagePrio.NOTICE, tags)

