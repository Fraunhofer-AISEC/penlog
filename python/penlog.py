# SPDX-License-Identifier: Apache-2.0

import inspect
import json
import os
import socket
import sys
from datetime import datetime
from enum import Enum, IntEnum
from typing import Dict, List, TextIO, Optional


# TODO: minimize! This is not intended like this.
class MessageType(str, Enum):
    ERROR = "error"
    WARNING = "warning"
    INFO = "info"
    DEBUG = "debug"
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


def _get_line_number(depth: int) -> str:
    stack = inspect.stack()
    frame = stack[depth]
    return f'{frame.filename}:{frame.lineno}'


class Logger:
    def __init__(self, component: str = "root", timefmt: str = '%c',
                 flush: bool = False, file_: TextIO = sys.stderr):
        self.host = socket.gethostname()
        self.component = component
        self.flush = flush
        self.file = file_

    def _log(self, msg: Dict, depth: int) -> None:
        msg["component"] = self.component
        msg["host"] = self.host
        msg["timestamp"] = datetime.now().isoformat()
        if os.environ.get("PENLOG_LINES"):
            msg["line"] = _get_line_number(depth)
        print(json.dumps(msg), file=self.file, flush=self.flush)

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

    def log_summary(self, data: str, tags: Optional[List[str]] = None) -> None:
        self._log_msg(data, MessageType.SUMMARY, MessagePrio.NOTICE, tags)


# This is the module level default logging emulation.

def _log(msg: Dict, depth: int) -> None:
    msg["component"] = "root"
    msg["host"] = socket.gethostname()
    msg["timestamp"] = datetime.now().isoformat()
    if os.environ.get("PENLOG_LINES"):
        msg["line"] = _get_line_number(depth)
    print(json.dumps(msg), file=sys.stderr, flush=True)


def _log_msg(data: str, type_: MessageType = MessageType.MESSAGE,
             prio: MessagePrio = MessagePrio.INFO,
             tags: Optional[List[str]] = None) -> None:
    msg = {
        'type': type_,
        'priority': prio,
        'data': data,
    }
    if tags:
        msg['tags'] = tags
    _log(msg, 4)


def log_msg(data: str, type_: MessageType = MessageType.MESSAGE,
            prio: MessagePrio = MessagePrio.INFO,
            tags: Optional[List[str]] = None) -> None:
    msg = {
        'type': type_,
        'priority': prio,
        'data': data,
    }
    if tags:
        msg['tags'] = tags
    _log(msg, 3)


def log_preamble(data: str) -> None:
    _log_msg(data, MessageType.PREAMBLE, MessagePrio.NOTICE)


def log_read(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.READ, MessagePrio.DEBUG, tags)


def log_write(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.WRITE, MessagePrio.DEBUG, tags)


def log_debug(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.MESSAGE, MessagePrio.DEBUG, tags)


def log_info(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.MESSAGE, MessagePrio.INFO, tags)


def log_notice(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.MESSAGE, MessagePrio.NOTICE, tags)


def log_warning(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.MESSAGE, MessagePrio.WARNING, tags)


def log_error(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.MESSAGE, MessagePrio.ERROR, tags)


def log_summary(data: str, tags: Optional[List[str]] = None) -> None:
    _log_msg(data, MessageType.SUMMARY, MessagePrio.NOTICE, tags)
