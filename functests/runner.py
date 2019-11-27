#!/usr/bin/env python3

import contextlib
import json
import os
import stat
import subprocess
import sys
import threading
import time


_ATTEMPT_MAX = 30
_ATTEMPT_DELAY = 10  # seconds

# we try hard to use only the stdlib, hence we don't use the
# rest client package.
class OC:
    def __init__(self, kubeconfig):
        self._kubeconfig = kubeconfig
        self._env = {
            'PATH': os.environ['PATH'],
            'KUBECONFIG': self._kubeconfig,
        }

    def pods(self, namespace=None):
        ns = ['--all-namespaces'] if namespace is None else ['-n', namespace]
        cmd = [
            'oc', 'get', 'pods', '-o', 'json'
        ]
        cmd.extend(ns)
        cmdline = ' '.join(cmd)
        res = subprocess.run(
            cmdline,
            shell=True,
            capture_output=True,
            env=self._env,
        )
        if res.returncode != 0:
            raise RuntimeError('[%s] failed: %s' % (cmdline, res.returncode))
        return json.loads(res.stdout)


class Cluster:
    # TODO: ENUM
    FAILED = -1
    STOPPED = 0
    STARTING = 1
    RUNNING = 2
    STOPPING = 3

    def __init__(self, provider, basedir):
        self._status = self.STOPPED
        self._provider = provider
        self._basedir = basedir
        self._proc = None
        self._env = make_env(
            PATH="%s:%s" % (
                os.path.join(basedir, "bin"),
                os.environ["PATH"]
            ),
            KUBEVIRTCI_RUNTIME='podman',
            KUBEVIRT_PROVIDER=self._provider,
        )

    @property
    def status(self):
        return self._status

    @classmethod
    def setup(cls, provider, basedir):
        if not os.access(
                os.path.join(
                    basedir,
                    "kubevirtci/cluster-up/cluster",
                    provider
                ),
                os.R_OK|os.X_OK
        ):
            raise RuntimeError("unknown provider %s", provider)

        if not os.access(
                os.path.join(basedir, "bin", "pack8s"),
                os.R_OK|os.X_OK
        ):
            raise RuntimeError("missing testing binary pack8s in %s" % (
                os.path.join(basedir, "bin")))

        return cls(provider, basedir)

    def start(self, timeout):
        if self._lock_file_exists():
            raise ValueError("%s already running" % self._provider)
        self._status = self.STARTING
        self._lock_file_create()
        with open(self._provider_log('up'), 'wt') as logfile:
            try:
                self._proc = subprocess.Popen(
                    self._call_make('cluster-up'),
                    shell=True,
                    text=True,
                    stdout=logfile,
                    env=self._env,
                )
            except Exception as exc:  # TODO: too broad
                self._status = self.FAILED
                self._proc = None
                self._lock_file_delete()
                raise
        self._wait_for(self.RUNNING, timeout)

    def stop(self, timeout):
        if not self._lock_file_exists():
            raise ValueError("%s not running" % self._provider)
        self._status = self.STOPPING
        with open(self._provider_log('down'), 'wt') as logfile:
            try:
                self._proc = subprocess.Popen(
                    self._call_make('cluster-down'),
                    shell=True,
                    text=True,
                    stdout=logfile,
                    env=self._env,
                )
            except Exception:
                self._status = self.FAILED
                raise
        self._wait_for(self.STOPPED, timeout)
        self._lock_file_delete()

    def kubeconfig(self):
        if not self._lock_file_exists():
            raise ValueError("%s not running" % self._provider)
        res = subprocess.run(
            os.path.join(self._basedir,
                "kubevirtci/cluster-up/kubeconfig.sh"
            ),
            capture_output=True,
            text=True,
            env=self._env,
        )
        # we don't want the trailing '\n' to end up into commandline
        # or env vars - that will lead to subtle bugs (learned the hard way)
        return res.stdout.strip()

    # helpers

    def _wait_for(self, status, timeout):
        self._check_legal(self._status, status)
        # TODO: what if self._proc is None?
        retcode = None
        try:
            retcode = self._proc.wait(timeout)
            # TODO: handle return code
        except subprocess.TimeoutExpired:
            self._proc.terminate()
            self._proc.wait()
            raise
        finally:
            # TODO: close?
            self._proc = None
        self._status = status if retcode == 0 else self.FAILED

    def _check_legal(self, current, desired):
        if current == self.STARTING and desired == self.RUNNING:
            return
        if current == self.STOPPING and desired == self.STOPPED:
            return
        raise ValueError("invalid transition: %s -> %s" % (current, desired))

    def _call_make(self, target):
        cmd = [
            "make",
            "-C",
            os.path.join(self._basedir, "kubevirtci"),
            target
        ]
        return " ".join(cmd)

    def _provider_log(self, suffix):
        return os.path.join(
            self._basedir, "%s-%s.log" % (self._provider, suffix)
        )

    def _lock_file_path(self, suffix=''):
        return os.path.join(
            self._basedir,
            "run-%s%s%s.lock" % (
                self._provider, '-' if suffix else '', suffix
            )
        )

    def _lock_file_exists(self):
        return os.access(self._lock_file_path(), os.R_OK)

    def _lock_file_create(self):
        return open(self._lock_file_path(), "wt")

    def _lock_file_delete(self):
        os.remove(self._lock_file_path())


def make_env(**kwargs):
    env = os.environ.copy()
    env.update(**kwargs)
    return env


@contextlib.contextmanager
def running_cluster(provider, basedir, timeout):
    cl = Cluster.setup(provider, basedir)
    cl.start(timeout)
    if cl.status != Cluster.RUNNING:
        raise RuntimeError("cluster start failed")

    try:
        yield OC(cl.kubeconfig())
    finally:
        cl.stop(timeout)
        pass


def run(provider, basedir, timeout, checkfn):
    with running_cluster(provider, basedir, timeout) as oc:
        try:
            return checkfn(oc.pods())
        except Exception as exc:
            sys.stderr.write("%s\n" % exc)
            return -1


def check_apiserver(data):
    pods = data['items']
    for pod in pods:
        name = pod['metadata']['name']
        status = pod['status']['phase']
        if 'apiserver' in name and status.lower() == 'running':
            return 0
    return -1


def _main():
    if len(sys.argv) not in (2,3,):
        sys.stderr.write("usage: %s provider [test-dir]\n" % sys.argv[0])
        sys.exit(1)
    if len(sys.argv) == 3:
        basedir_src = 'command line'
        basedir = sys.argv[2]
    else:
        basedir_src = 'environment variable'
        basedir = os.environ.get('PACK8S_FUNCTEST_DIR')
        if basedir is None:
            sys.stderr.write(
                "%s: no test-dir given,"
                " not env var PACK8S_FUNCTEST_DIR set\n" % sys.argv[0])
            sys.exit(1)
    print('test dir: %s from %s' % (basedir, basedir_src))
    provider = sys.argv[1]
    timeout = 300  # seconds
    return run(provider, basedir, timeout, check_apiserver)


if __name__ == "__main__":
    sys.exit(_main())
