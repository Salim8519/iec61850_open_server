"""Live, write-through views of the engine's in-memory project.

In process, `projects.project` and the IED dicts `add_template` returns are the service's own
objects: later reads see later changes and `ied.update(ip=...)` edits the project. These
views keep that meaning: every read fetches the engine's current project and walks a path
whose list steps are ids ({"id": ...}), and writes go to the parity inspect API.
"""
from parity.errors import InProcessOnly
from parity.inspect_ops import walk


def _step(key, value):
    return {"id": value["id"]} if isinstance(value, dict) and "id" in value else key


def wrap(value, root, path):
    if isinstance(value, dict):
        return LiveDict(root, path)
    if isinstance(value, list):
        return LiveList(root, path)
    return value


class LiveRoot:
    """Fetches the whole project once per read and forwards writes."""

    def __init__(self, engine):
        self.engine = engine

    def fetch(self):
        return self.engine.parity("GET", "project/raw")

    def set(self, path, key, value):
        self.engine.parity("POST", "project/set", {"path": path, "key": key, "value": value})

    def update(self, path, values):
        self.engine.parity("POST", "project/update", {"path": path, "values": values})


class LiveDict(dict):
    def __init__(self, root, path):
        super().__init__()
        self._root, self._path = root, path

    def _fresh(self):
        value = walk(self._root.fetch(), self._path)
        dict.clear(self)
        dict.update(self, value)
        return value

    def __getitem__(self, key):
        return wrap(self._fresh()[key], self._root, self._path + [key])

    def get(self, key, default=None):
        return self[key] if key in self._fresh() else default

    def __contains__(self, key):
        return key in self._fresh()

    def __iter__(self):
        return iter(list(self._fresh()))

    def __len__(self):
        return len(self._fresh())

    def keys(self):
        return self._fresh().keys()

    def values(self):
        return [self[key] for key in list(self._fresh())]

    def items(self):
        return [(key, self[key]) for key in list(self._fresh())]

    def copy(self):
        return dict(self._fresh())

    def __eq__(self, other):
        return self._fresh() == other

    def __ne__(self, other):
        return not self == other

    __hash__ = None

    def __repr__(self):
        return repr(self._fresh())

    def __setitem__(self, key, value):
        self._root.set(self._path, key, value)

    def update(self, *args, **values):
        self._root.update(self._path, dict(*args, **values))

    def __delitem__(self, key):
        raise InProcessOnly(f"del project{self._path}[{key!r}]")

    pop = popitem = setdefault = clear = __delitem__


class LiveList(list):
    def __init__(self, root, path):
        super().__init__()
        self._root, self._path = root, path

    def _fresh(self):
        value = walk(self._root.fetch(), self._path)
        list.clear(self)
        list.extend(self, value)
        return value

    def __getitem__(self, index):
        fresh = self._fresh()
        if isinstance(index, slice):
            return [wrap(item, self._root, self._path + [_step(position, item)])
                    for position, item in zip(range(len(fresh))[index], fresh[index])]
        value = fresh[index]
        return wrap(value, self._root, self._path + [_step(index % len(fresh), value)])

    def __iter__(self):
        fresh = self._fresh()
        return iter([wrap(item, self._root, self._path + [_step(index, item)]) for index, item in enumerate(fresh)])

    def __len__(self):
        return len(self._fresh())

    def __contains__(self, item):
        return item in self._fresh()

    def index(self, item, *bounds):
        return self._fresh().index(item, *bounds)

    def count(self, item):
        return self._fresh().count(item)

    def copy(self):
        return list(self)

    def __reversed__(self):
        return reversed(list(self))

    def __add__(self, other):
        return list(self) + list(other)

    def __eq__(self, other):
        return self._fresh() == other

    def __ne__(self, other):
        return not self == other

    __hash__ = None

    def __repr__(self):
        return repr(self._fresh())

    def _refuse(self, *_args, **_kwargs):
        raise InProcessOnly(f"mutating list project{self._path}")

    append = extend = insert = remove = pop = sort = reverse = __setitem__ = __delitem__ = _refuse
