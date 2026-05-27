#!/usr/bin/env python3
"""
Remote executor via SSH/SFTP using paramiko.

Usage as CLI:
    python remote_exec.py exec "helm list -A"
    python remote_exec.py upload ./local.tar /tmp/remote.tar
    python remote_exec.py download /tmp/remote.log ./local.log

Usage as module:
    from remote_exec import RemoteExecutor

    with RemoteExecutor(host="120.55.84.39", user="root", password="...") as e:
        exit_code, stdout, stderr = e.exec("ls /tmp")
        e.upload("local.tar", "/tmp/remote.tar")
        e.download("/tmp/remote.tar", "local.tar")

Connection info priority: CLI args > env vars (REMOTE_HOST, REMOTE_USER, REMOTE_PASSWORD) > defaults
"""

import os
import sys
import argparse
import stat
from contextlib import contextmanager

try:
    import paramiko
except ImportError:
    sys.exit("paramiko is required. Install with: pip install paramiko")


class RemoteExecutorError(Exception):
    """Raised when a remote operation fails."""
    pass


class RemoteExecutor:
    """SSH/SFTP remote command executor and file transfer client."""

    def __init__(
        self,
        host="120.55.84.39",
        port=22,
        user="root",
        password="",
        connect_timeout=15,
    ):
        self.host = host
        self.port = port
        self.user = user
        self.password = password
        self.connect_timeout = connect_timeout
        self._client = None
        self._sftp = None

    def _ensure_connected(self):
        if self._client is not None:
            return
        client = paramiko.SSHClient()
        client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        try:
            client.connect(
                hostname=self.host,
                port=self.port,
                username=self.user,
                password=self.password,
                timeout=self.connect_timeout,
                allow_agent=False,
                look_for_keys=False,
            )
        except Exception as e:
            raise RemoteExecutorError(
                f"Failed to connect to {self.user}@{self.host}:{self.port}: {e}"
            ) from e
        self._client = client

    def _ensure_sftp(self):
        self._ensure_connected()
        if self._sftp is not None:
            return self._sftp
        self._sftp = self._client.open_sftp()
        return self._sftp

    def exec(self, command, stream=False, timeout=None):
        """
        Execute a command on the remote host.

        Args:
            command: Shell command to execute.
            stream: If True, yield (channel, stream_name, line) tuples for real-time output.
                    If False, return (exit_code, stdout, stderr).
            timeout: Command timeout in seconds. None = no timeout.

        Returns:
            If stream=False: (exit_code: int, stdout: str, stderr: str)
            If stream=True: generator yielding (channel, stream_name: str, line: str)
        """
        self._ensure_connected()
        chan = self._client.get_transport().open_session(timeout=timeout)
        chan.set_combine_stderr(False)
        chan.exec_command(command)

        if stream:
            return self._stream_output(chan)
        else:
            return self._collect_output(chan, timeout)

    def _stream_output(self, chan):
        """Generator that yields (channel, stream_name, line) for real-time output."""
        stdout_buf = b""
        stderr_buf = b""

        while not chan.exit_status_ready():
            if chan.recv_ready():
                data = chan.recv(4096)
                if data:
                    stdout_buf += data
                    while b"\n" in stdout_buf:
                        line, stdout_buf = stdout_buf.split(b"\n", 1)
                        yield (chan, "stdout", line.decode("utf-8", errors="replace"))
            if chan.recv_stderr_ready():
                data = chan.recv_stderr(4096)
                if data:
                    stderr_buf += data
                    while b"\n" in stderr_buf:
                        line, stderr_buf = stderr_buf.split(b"\n", 1)
                        yield (chan, "stderr", line.decode("utf-8", errors="replace"))

        # Drain remaining
        while chan.recv_ready():
            stdout_buf += chan.recv(4096)
        while chan.recv_stderr_ready():
            stderr_buf += chan.recv_stderr(4096)

        if stdout_buf:
            yield (chan, "stdout", stdout_buf.decode("utf-8", errors="replace"))
        if stderr_buf:
            yield (chan, "stderr", stderr_buf.decode("utf-8", errors="replace"))

    def _collect_output(self, chan, timeout):
        """Collect all output, return (exit_code, stdout, stderr)."""
        stdout_lines = []
        stderr_lines = []
        for _, stream_name, line in self._stream_output(chan):
            if stream_name == "stdout":
                stdout_lines.append(line)
            else:
                stderr_lines.append(line)
        exit_code = chan.recv_exit_status()
        return (exit_code, "\n".join(stdout_lines), "\n".join(stderr_lines))

    def upload(self, local_path, remote_path):
        """Upload a file to the remote host via SFTP."""
        sftp = self._ensure_sftp()
        try:
            sftp.put(local_path, remote_path)
        except Exception as e:
            raise RemoteExecutorError(
                f"Failed to upload {local_path} -> {remote_path}: {e}"
            ) from e

    def download(self, remote_path, local_path):
        """Download a file from the remote host via SFTP."""
        sftp = self._ensure_sftp()
        try:
            sftp.get(remote_path, local_path)
        except Exception as e:
            raise RemoteExecutorError(
                f"Failed to download {remote_path} -> {local_path}: {e}"
            ) from e

    def close(self):
        if self._sftp is not None:
            self._sftp.close()
            self._sftp = None
        if self._client is not None:
            self._client.close()
            self._client = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
        return False


def _get_executor_from_env_or_args(args):
    """Build RemoteExecutor from CLI args merged with env vars."""
    host = args.host or os.environ.get("REMOTE_HOST", "120.55.84.39")
    port = args.port or int(os.environ.get("REMOTE_PORT", "22"))
    user = args.user or os.environ.get("REMOTE_USER", "root")
    password = args.password or os.environ.get("REMOTE_PASSWORD", "")
    return RemoteExecutor(host=host, port=port, user=user, password=password)


def cmd_exec(args):
    executor = _get_executor_from_env_or_args(args)
    with executor:
        if args.stream:
            for _, stream_name, line in executor.exec(args.command, stream=True):
                if stream_name == "stdout":
                    print(line)
                else:
                    print(line, file=sys.stderr)
        else:
            exit_code, stdout, stderr = executor.exec(args.command)
            if stdout:
                print(stdout)
            if stderr:
                print(stderr, file=sys.stderr)
            sys.exit(exit_code)


def cmd_upload(args):
    executor = _get_executor_from_env_or_args(args)
    with executor:
        executor.upload(args.local_path, args.remote_path)
        print(f"Uploaded {args.local_path} -> {args.remote_path}")


def cmd_download(args):
    executor = _get_executor_from_env_or_args(args)
    with executor:
        executor.download(args.remote_path, args.local_path)
        print(f"Downloaded {args.remote_path} -> {args.local_path}")


def main():
    parser = argparse.ArgumentParser(description="Remote executor via SSH/SFTP")
    parser.add_argument("--host", help="Remote host (env: REMOTE_HOST)")
    parser.add_argument("--port", type=int, help="SSH port (env: REMOTE_PORT, default: 22)")
    parser.add_argument("--user", help="SSH user (env: REMOTE_USER, default: root)")
    parser.add_argument("--password", help="SSH password (env: REMOTE_PASSWORD)")

    subparsers = parser.add_subparsers(dest="action", required=True)

    exec_parser = subparsers.add_parser("exec", help="Execute remote command")
    exec_parser.add_argument("command", help="Command to execute")
    exec_parser.add_argument("--stream", action="store_true", help="Stream output in real-time")
    exec_parser.set_defaults(func=cmd_exec)

    upload_parser = subparsers.add_parser("upload", help="Upload a file")
    upload_parser.add_argument("local_path", help="Local file path")
    upload_parser.add_argument("remote_path", help="Remote destination path")
    upload_parser.set_defaults(func=cmd_upload)

    download_parser = subparsers.add_parser("download", help="Download a file")
    download_parser.add_argument("remote_path", help="Remote file path")
    download_parser.add_argument("local_path", help="Local destination path")
    download_parser.set_defaults(func=cmd_download)

    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()
