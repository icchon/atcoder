import math
from collections import defaultdict
import sys
import heapq
from collections import Counter
from collections import deque
import random
import os
# import copy 
# import itertools
import bisect
# import  sortedcontainers
# from atcoder.segtree import SegTree
# from atcoder.dsu import DSU

ONLINE_JUDGE = os.environ.get("ATCODER") == "1"
if not ONLINE_JUDGE: import icecream
def get_ic(is_debug): return icecream.ic if is_debug else lambda *args: args[0] if len(args) == 1 else (None if len(args) == 0 else (args))
ic = get_ic(not ONLINE_JUDGE)

sys.setrecursionlimit(1000000000)
# "a" + "b" <=> "".join(["a", "b"])　再帰　はCPython
input = sys.stdin.readline
MODINT = 998244353
INF=float("inf")
MINF=float("-inf")

class SqrtList:
    BUCKET_SIZE = 1000
    def __init__(self, xs=None):
        xs = list([] if xs is None else xs)
        self.buckets = [xs[i:i + self.BUCKET_SIZE] for i in range(0, len(xs), self.BUCKET_SIZE)] if len(xs) > 0 else [[]]
    def insert(self, i, v):
        cur = 0
        for idx, b in enumerate(self.buckets):
            if cur + len(b) >= i:
                b.insert(i - cur, v)
                if len(b) > 2 * self.BUCKET_SIZE:
                    mid = len(b) // 2
                    self.buckets.insert(idx + 1, b[mid:])
                    del b[mid:]
                return
            cur += len(b)
        self.buckets[-1].append(v)
    def pop(self, i=-1):
        if i < 0: i += len(self)
        if not (0 <= i < len(self)): raise IndexError()
        cur = 0
        for idx, b in enumerate(self.buckets):
            if cur + len(b) > i:
                val = b.pop(i - cur)
                if len(b) == 0 and len(self.buckets) > 1: del self.buckets[idx]
                return val
            cur += len(b)
    def __iter__(self):
        for b in self.buckets: 
            for x in b: yield x
    def __getitem__(self, i):
        if i < 0: i += len(self)
        cur = 0
        for b in self.buckets:
            if cur + len(b) > i: return b[i - cur]
            cur += len(b)
        raise IndexError()
    def __delitem__(self, i): self.pop(i)
    def __setitem__(self, i, v):
        if i < 0: i += len(self)
        cur = 0
        for b in self.buckets:
            if cur + len(b) > i:
                b[i - cur] = v
                return
            cur += len(b)
        raise IndexError()
    def __len__(self): return sum(len(b) for b in self.buckets)
    def __str__(self): return "".join(map(str, self.buckets))
    __repr__ = __str__

class TreapList:
    def __init__(self, xs=None):
        self.L = [0]   # 左の子
        self.R = [0]   # 右の子
        self.V = [None]  # 値
        self.P = [0.0]   # 優先度
        self.S = [0]   # 部分木のサイズ
        self.root = 0
        xs = [] if xs is None else list(xs)
        if xs: self.root = self._build(xs)
    def _build(self, xs):
        n = len(xs)
        base = len(self.V)
        L, R, S, P = self.L, self.R, self.S, self.P
        L.extend([0] * n); R.extend([0] * n); S.extend([0] * n); P.extend([0.0] * n)
        self.V.extend(xs)
        def rec(lo, hi):
            if lo >= hi:
                return 0
            mid = (lo + hi) // 2
            t = base + mid
            L[t] = rec(lo, mid)
            R[t] = rec(mid + 1, hi)
            S[t] = hi - lo
            return t
        root = rec(0, n)
        pri = sorted((random.random() for _ in range(n)), reverse=True)
        q = [root]
        k = 0
        while k < len(q):
            t = q[k]
            P[t] = pri[k]
            k += 1
            if L[t]: q.append(L[t])
            if R[t]: q.append(R[t])
        return root
    def _new(self, v):
        self.L.append(0); self.R.append(0); self.V.append(v)
        self.P.append(random.random()); self.S.append(1)
        return len(self.V) - 1
    def _update(self, t): self.S[t] = self.S[self.L[t]] + self.S[self.R[t]] + 1
    def _split(self, t, k):
        if not t: return 0, 0
        L, R, S = self.L, self.R, self.S
        if S[L[t]] >= k:
            a, b = self._split(L[t], k)
            L[t] = b
            self._update(t)
            return a, t
        else:
            a, b = self._split(R[t], k - S[L[t]] - 1)
            R[t] = a
            self._update(t)
            return t, b
    def _merge(self, a, b):
        if not a or not b:return a or b
        if self.P[a] > self.P[b]:
            self.R[a] = self._merge(self.R[a], b)
            self._update(a)
            return a
        else:
            self.L[b] = self._merge(a, self.L[b])
            self._update(b)
            return b
    def _norm(self, i):
        n = self.S[self.root]
        if i < 0:i += n
        if not (0 <= i < n): raise IndexError()
        return i
    def _find(self, i):
        L, R, S = self.L, self.R, self.S
        t = self.root
        while True:
            ls = S[L[t]]
            if i < ls:
                t = L[t]
            elif i == ls:
                return t
            else:
                i -= ls + 1
                t = R[t]

    def insert(self, i, v):
        n = self.S[self.root]
        if i < 0: i = max(0, i + n)
        i = min(i, n)
        a, b = self._split(self.root, i)
        self.root = self._merge(self._merge(a, self._new(v)), b)
    def append(self, v): self.root = self._merge(self.root, self._new(v))
    def pop(self, i=-1):
        if self.S[self.root] == 0: raise IndexError()
        i = self._norm(i)
        a, b = self._split(self.root, i)
        m, c = self._split(b, 1)
        self.root = self._merge(a, c)
        return self.V[m]
    def __getitem__(self, i): return self.V[self._find(self._norm(i))]
    def __setitem__(self, i, v): self.V[self._find(self._norm(i))] = v
    def __delitem__(self, i): self.pop(i)
    def __len__(self): return self.S[self.root]
    def __iter__(self):
        L, R, V = self.L, self.R, self.V
        stack, t = [], self.root
        while stack or t:
            while t:
                stack.append(t)
                t = L[t]
            t = stack.pop()
            yield V[t]
            t = R[t]
    def __str__(self): return str(list(self))
    __repr__ = __str__

class F:
    @staticmethod
    def LAMBDA(f): return lambda x:f(x)
    @staticmethod
    def hd(xs): return None if F.empty(xs) else xs[0]
    @staticmethod
    def tail(xs): return None if F.empty(xs) else xs[-1]
    @staticmethod
    def head_tails(xs):
        if len(xs) == 0: return ([],[])
        if len(xs) == 1: return (xs[0],[])
        else: return (xs[0],xs[1:])
    @staticmethod
    def heads_tail(xs):
        if len(xs) == 0: return ([], [])
        if len(xs) == 1: return ([], xs[-1])
        else: return (xs[:-1], xs[-1])
    @staticmethod
    def map_curry(f): return lambda xs:map(f,xs)
    @staticmethod
    def revlist(xs): return F.compose(list, reversed)(xs)
    @staticmethod
    def fst(xs): return xs[0]
    @staticmethod
    def snd(xs): return xs[1]
    @staticmethod
    def mapi(f, xs):return [f(i,x) for i,x in enumerate(xs)]
    @staticmethod
    def maplist(f, xs): return F.compose(list, F.map_curry(f))(xs)
    @staticmethod
    def inc(x): return x+1
    @staticmethod
    def dec(x): return x-1
    @staticmethod
    def compose(f, g): return lambda x:f(g(x))
    @staticmethod
    def fold_left(f, xs, acc0):
        acc = acc0
        for x in xs: acc = f(acc, x)
        return acc
    @staticmethod
    def fold_left2(f, xs, y2, acc0):
        acc = acc0
        for x,y in zip(xs, y2): acc = f(acc, x, y)
        return acc
    @staticmethod
    def any(xs, f=bool):
        for x in xs:
            if f(x): return True
        return False
    @staticmethod
    def all(xs, f=bool):
        for x in xs:
            if not f(x):return False
        return True
    @staticmethod
    def identity(x): return x
    @staticmethod
    def argmax(xs, f=lambda x:x):
        res, cand = None, float("-inf")
        for i,fx in F.compose(enumerate, F.map_curry(f))(xs):
            if cand < fx: (cand, res) = (fx, i)
        return res
    @staticmethod
    def empty(xs): return len(xs) == 0
    @staticmethod
    def filter(f, xs): return [x for x in xs if f(x)]
    @staticmethod
    def find(f, xs):
        res =  F.filter(f, xs)
        return None if F.empty(res) else F.hd(res)
class UnionFind():
    def __init__(self, n): self.n, self.parents = n, [-1]*n
    def find(self, x):
        if self.parents[x] < 0: return x
        self.parents[x] = self.find(self.parents[x])
        return self.parents[x]
    def union(self, x, y):
        x, y = self.find(x), self.find(y)
        if x == y: return
        if self.parents[x] > self.parents[y]: x, y = y, x
        self.parents[x] += self.parents[y]
        self.parents[y] = x
    def size(self, x): return -self.parents[self.find(x)]
    def same(self, x, y): return self.find(x) == self.find(y)
    def members(self, x):
        root = self.find(x)
        return [i for i in range(self.n) if self.find(i) == root]
    def members_count(self):
        lens = [0]*self.n
        for x in range(self.n): lens[self.find(x)] += 1
        return [lens[self.find(x)] for x in range(self.n)]
    def roots(self): return [i for i, x in enumerate(self.parents) if x < 0]
    def group_count(self): return len(self.roots())
    def all_group_members(self):
        group_members = defaultdict(list)
        for member in range(self.n): group_members[self.find(member)].append(member)
        return group_members
    def __str__(self): return '\n'.join(f'{r}: {m}' for r, m in self.all_group_members().items())
DX2_VERTICAL = [(1, 0), (0, 1), (-1, 0), (0, -1)]
DX2_DIAGONAL = [(1, 1), (-1, 1), (-1, -1), (1, -1)]
DX3_VERTICAL = [(1, 0, 0), (0, 1, 0), (-1, 0, 0), (0, -1, 0), (0, 0, 1), (0, 0, -1)]
DX3_DIAGONAL = [(1, 1, 1), (-1, 1, 1), (-1, -1, 1), (1, -1, 1), (1, 1, -1), (-1, 1, -1), (-1, -1, -1), (1, -1, -1)]
class Vec2:
    @staticmethod 
    def add(v1, v2): return tuple(map(lambda x,y:x+y, v1, v2))
    @staticmethod
    def neg(v): return tuple(map(lambda x:-x, v))
    @staticmethod
    def cross(v1, v2):
        (x1, y1) = v1
        (x2, y2) = v2
        return (x1*y2 - x2*y1)
    def sub(v1, v2): return tuple(map(lambda x,y:x-y, v1, v2))
    @staticmethod
    def mid(v1, v2): return tuple(map(lambda x,y:(x+y)/2, v1, v2))
    @staticmethod
    def mul(s, v): return tuple(map(lambda x:s*x, v))
    @staticmethod
    def dot(v1, v2): return sum(map((lambda x,y:x*y, v1, v2)))
    @staticmethod
    def norm2(v): return Vec2.dot(v, v)
    @staticmethod
    def normal(v): return (-F.snd(v), F.fst(v))
    @staticmethod
    def is_para(v1, v2): return (F.fst(v1)*F.snd(v2) == F.snd(v1)*F.fst(v2))
class Mat2:
    @staticmethod
    def add(m1, m2): return [[m1[i][j]+m2[i][j] for j in range(len(m1[0]))] for i in range(len(m1))]
    @staticmethod
    def zero(n): return [[0]*n for _ in range(n)]
    @staticmethod
    def neg(m): return [[-m[i][j] for j in range(len(m[0]))] for i in range(len(m))]
    @staticmethod
    def sub(m1,m2):return Mat2.add(m1, Mat2.neg(m2))
    @staticmethod
    def one(n): return [[(1 if i == j else 0) for j in range(n)] for i in range(n)]
    @staticmethod
    def mul(m1, m2, mod=None):
        H, K, W = len(m1), len(m2), len(m2[0])
        assert len(m1[0]) == K
        res = [[0] * W for _ in range(H)]
        for i in range(H):
            ri = res[i]
            for k in range(K):
                a = m1[i][k]
                if a == 0:
                    continue
                m2k = m2[k]
                for j in range(W):
                    ri[j] += a * m2k[j]
            if mod is not None:
                res[i] = [x % mod for x in ri]
        return res
    @staticmethod
    def pow(m, n, mod=None):
        res = Mat2.one(len(m))
        while n:
            if n & 1:
                res = Mat2.mul(res, m, mod)
            m = Mat2.mul(m, m, mod)
            n >>= 1
        return res

class Graph:
    #有向グラフの閉路
    @staticmethod
    def is_ditected_graph_closed(N_vertex, edges):
        graph = [[] for _ in range(N_vertex)]
        for (s, g) in edges: graph[s].append(g)
        def dfs(v, seen, finished):
            seen[v] = True
            for v_next in graph[v]:
                if finished[v_next]: continue
                if seen[v_next]: return True
                if dfs(v_next, seen, finished): return True
            finished[v] = True
            return False
        N = len(graph)
        seen = [False]*N
        finished = [False]*N
        for v in range(N): 
            if dfs(v, seen, finished): return True
        return False
    #無向グラフの閉路 
    @staticmethod
    def is_unditected_graph_closed(N_vertex, edges):
        uni = UnionFind(N_vertex)
        for (v1, v2) in edges:
            if uni.same(v1, v2): return True
            uni.union(v1, v2)
        return False
    @staticmethod
    def topological_sort(G):
        n = len(G)
        # 入次数 indeg[v]: u -> v となるような数
        indeg = [0]*n
        for u in range(n): 
            for v in G[u]: indeg[v] += 1
        pq = Heapq()
        for u in range(n):
            if indeg[u] == 0: pq.push(u)
        order = []
        while(len(pq) > 0):
            u = pq.pop()
            order.append(u)
            for v in G[u]:
                indeg[v] -= 1
                if indeg[v] == 0: pq.push(v)
        return order
    @staticmethod
    def bellman_ford(n, edges, s):
        d = [INF]*n
        d[s] = 0
        for _ in range(n-1):
            updated = False
            for a,b,c in edges:
                if d[a] != INF and d[a] + c < d[b]:
                    d[b] = d[a] + c
                    updated = True
            if (not updated):break
        for _ in range(n):
            for a,b,c in edges:
                if d[a] != INF and d[a] + c < d[b]:d[b] = MINF
        return d
class Heapq:
    class Item:
        __slots__ = ("val", "greater")
        def __init__(self, val, greater): self.val, self.greater = val, greater
        def __lt__(self, other): return self.val > other.val if self.greater else self.val < other.val
        def __repr__(self): return str(self.val)
    def __init__(self, greater=False): self.xs, self.greater = [], greater
    def push(self, x): heapq.heappush(self.xs, self.Item(x, self.greater))
    def pop(self): return heapq.heappop(self.xs).val
    def __len__(self): return len(self.xs)
    def __bool__(self): return bool(self.xs)
    def top(self): return self.xs[0].val
    def __repr__(self): return str(self.xs)
class Grid:
    @staticmethod
    def transform(grid, h, w, f):
        res = Grid.grid_hw(h, w, None)
        for i in range(h):
            for j in range(w):
                (ni, nj) = f(i, j)
                res[i][j] = grid[ni][nj]
        return res
    @staticmethod
    def transpose(grid):
        h,w = len(grid), len(F.hd(grid))
        assert h == w
        return Grid.transform(grid, h, w, lambda i,j:(j,i))
    @staticmethod
    def rotate90(grid): return Grid.transform(grid, len(F.hd(grid)), len(grid), lambda i,j:(j, len(F.hd(grid)) - i - 1))
    @staticmethod
    def symmetric_y(grid): return Grid.transform(grid, len(grid), len(F.hd(grid)), lambda i,j:(i, len(F.hd(grid)) - j - 1))
    @staticmethod
    def symmetric_x(grid): return Grid.transform(grid, len(grid), len(F.hd(grid)), lambda i,j:(len(grid) - i - 1, j))
    @staticmethod
    def print(arrs):
        print("-"*len(F.hd(arrs))*2)
        for arr in arrs: print(*arr)
        print("-"*len(F.hd(arrs))*2)
    @staticmethod
    def is_inner(hw, xy):
        h,w = hw
        x,y = xy
        return (0 <= x < h) and (0 <= y < w)
    @staticmethod
    def grid_hw(h, w, v): return [[v]*w for _ in range(h)]
    @staticmethod
    def slice(grid, upleft, bottomright):
        sx, sy = upleft
        tx, ty = bottomright
        return [grid[i][sy:ty+1] for i in range(sx, tx+1)]
    @staticmethod
    def equal(g1, g2):
        h1,w1, h2,w2 = len(g1), len(F.hd(g1)), len(g2), len(F.hd(g2))
        if not ((h1 == h2) and (w1 == w2)): return False
        for i in range(h1):
            for j in range(w1): 
                if g1[i][j] != g2[i][j]: return False
        return True
class Util:
    @staticmethod
    def run_length_encode(xs):
        res = []
        if len(xs) == 0: return res
        pre,cnt = xs[0], 1
        for x in xs[1:]:
            if x != pre:
                res.append((pre, cnt))
                pre, cnt = x, 1
            else: cnt += 1
        return res + [(pre, cnt)]
    @staticmethod
    def prime_factorization(n):
        x,cnt = n, defaultdict(int)
        for i in range(2, math.floor(math.sqrt(n)) + 1):
            if x % i != 0: continue
            c = 0
            while(x % i == 0): c,x = c + 1, x // i
            cnt[i] += c
            if x == 1: return cnt
        cnt[x] += 1
        return cnt
    @staticmethod
    def get_sieve(n):
        primes = [True] * (n + 1)
        primes[0] = False
        if n >= 1:primes[1] = False
        limit = int(n**0.5)
        for p in range(2, limit + 1):
            if primes[p]: 
                for np in range(p * p, n + 1, p): primes[np] = False
        return primes
    @staticmethod
    def primerange_gen(l, r):
        if l >= r or r <= 2: return
        sieve = Util.get_sieve(r - 1)
        for p in range(max(2,l), r): 
            if sieve[p]: yield p
    @staticmethod
    def split_step_k(xs, K, v=None):
        n = (K*math.ceil(len(xs)/K))
        ref = [v]*n
        for i in range(len(xs)): ref[i] = xs[i]
        return [[ref[offset+i*K] for i in range(n//K)] for offset in range(K)]
    @staticmethod
    def is_overlap(lr1, lr2):
        l1,r1 = lr1
        l2,r2 = lr2
        return max(l1,l2) <= min(r1, r2)
    @staticmethod
    def floor(a, b): return a//b
    @staticmethod
    def ceil(a, b): return -(-a//b)
    @staticmethod
    def compress(xs):
        d,rev_d = {},{}
        for i,x in enumerate(sorted(set(xs))): d[x], rev_d[i] = i, x
        return (d,rev_d)
class SegTree:
    """op: 二項演算, e: 単位元, v: 要素数 または 初期配列
    0-indexed、半開区間 [left, right)"""
    def __init__(self, op, e, v):
        self._op = op
        self._e = e
        if isinstance(v, int):
            v = [e] * v
        self._n = len(v)
        self._log = 0
        while (1 << self._log) < self._n:  # 2^log >= n となる最小の log
            self._log += 1
        self._size = 1 << self._log
        self._d = [e] * (2 * self._size)
        for i in range(self._n):
            self._d[self._size + i] = v[i]
        for i in range(self._size - 1, 0, -1):
            self._update(i)
    def set(self, p, x):
        """a[p] = x"""
        assert 0 <= p < self._n
        p += self._size
        self._d[p] = x
        for i in range(1, self._log + 1):
            self._update(p >> i)
    def get(self, p):
        """a[p]"""
        assert 0 <= p < self._n
        return self._d[p + self._size]
    def prod(self, left, right):
        """op(a[left], ..., a[right - 1])（left == right なら e）"""
        assert 0 <= left <= right <= self._n
        sml = self._e
        smr = self._e
        left += self._size
        right += self._size
        while left < right:
            if left & 1:
                sml = self._op(sml, self._d[left])
                left += 1
            if right & 1:
                right -= 1
                smr = self._op(self._d[right], smr)
            left >>= 1
            right >>= 1
        return self._op(sml, smr)

    def all_prod(self):
        """op(a[0], ..., a[n - 1]) """ 
        return self._d[1]
    def max_right(self, left, f):
        """f(op(a[left], ..., a[r - 1])) が True となる最大の r （f は単調）"""
        assert 0 <= left <= self._n
        assert f(self._e)
        if left == self._n:return self._n
        left += self._size
        sm = self._e
        first = True
        while first or (left & -left) != left:
            first = False
            while left % 2 == 0:
                left >>= 1
            if not f(self._op(sm, self._d[left])):
                while left < self._size:
                    left *= 2
                    if f(self._op(sm, self._d[left])):
                        sm = self._op(sm, self._d[left])
                        left += 1
                return left - self._size
            sm = self._op(sm, self._d[left])
            left += 1
        return self._n
    def min_left(self, right, f):
        """f(op(a[l], ..., a[right - 1])) が True となる最小の l（f は単調）"""
        assert 0 <= right <= self._n
        assert f(self._e)
        if right == 0:return 0
        right += self._size
        sm = self._e
        first = True
        while first or (right & -right) != right:
            first = False
            right -= 1
            while right > 1 and right % 2:
                right >>= 1
            if not f(self._op(self._d[right], sm)):
                while right < self._size:
                    right = 2 * right + 1
                    if f(self._op(self._d[right], sm)):
                        sm = self._op(self._d[right], sm)
                        right -= 1
                return right + 1 - self._size
            sm = self._op(self._d[right], sm)
        return 0
    def __str__(self): return str(self._d[self._size:self._size + self._n])
    __repr__ = __str__
    def _update(self, k):self._d[k] = self._op(self._d[2 * k], self._d[2 * k + 1])

class LazySegTree:
    """op, e,
    mapping(f, x): 更新 f をデータ x に作用させた結果,
    composition(f, g): 先に g、次に f を適用する更新（f ∘ g）, id_: 何もしない更新,
    v: 要素数 または 初期配列
    0-indexed、半開区間 [left, right)"""
    def __init__(self, op, e, mapping, composition, id_, v):
        self._op = op
        self._e = e
        self._mapping = mapping
        self._composition = composition
        self._id = id_
        if isinstance(v, int):
            v = [e] * v
        self._n = len(v)
        self._log = 0
        while (1 << self._log) < self._n:  # 2^log >= n となる最小の log
            self._log += 1
        self._size = 1 << self._log
        self._d = [e] * (2 * self._size)
        self._lz = [id_] * self._size
        for i in range(self._n):
            self._d[self._size + i] = v[i]
        for i in range(self._size - 1, 0, -1):
            self._update(i)
    def set(self, p, x):
        """a[p] = x"""
        assert 0 <= p < self._n
        p += self._size
        for i in range(self._log, 0, -1):
            self._push(p >> i)
        self._d[p] = x
        for i in range(1, self._log + 1):
            self._update(p >> i)
    def get(self, p):
        """a[p]"""
        assert 0 <= p < self._n
        p += self._size
        for i in range(self._log, 0, -1):
            self._push(p >> i)
        return self._d[p]
    def prod(self, left, right):
        """op(a[left], ..., a[right - 1])（left == right なら e）"""
        assert 0 <= left <= right <= self._n
        if left == right:
            return self._e
        left += self._size
        right += self._size
        for i in range(self._log, 0, -1):
            if ((left >> i) << i) != left:
                self._push(left >> i)
            if ((right >> i) << i) != right:
                self._push((right - 1) >> i)
        sml = self._e
        smr = self._e
        while left < right:
            if left & 1:
                sml = self._op(sml, self._d[left])
                left += 1
            if right & 1:
                right -= 1
                smr = self._op(self._d[right], smr)
            left >>= 1
            right >>= 1
        return self._op(sml, smr)
    def all_prod(self):
        """op(a[0], ..., a[n - 1])"""
        return self._d[1]
    def apply(self, left, right, f=None):
        """apply(l, r, f): [l, r) の全要素に f を作用させる
        apply(p, f)   : a[p] に f を作用させる"""
        if f is None:  # 1点への作用
            p, f = left, right
            assert 0 <= p < self._n
            p += self._size
            for i in range(self._log, 0, -1):
                self._push(p >> i)
            self._d[p] = self._mapping(f, self._d[p])
            for i in range(1, self._log + 1):
                self._update(p >> i)
            return
        assert 0 <= left <= right <= self._n
        if left == right:
            return
        left += self._size
        right += self._size
        for i in range(self._log, 0, -1):
            if ((left >> i) << i) != left:
                self._push(left >> i)
            if ((right >> i) << i) != right:
                self._push((right - 1) >> i)
        l2, r2 = left, right
        while left < right:
            if left & 1:
                self._all_apply(left, f)
                left += 1
            if right & 1:
                right -= 1
                self._all_apply(right, f)
            left >>= 1
            right >>= 1
        left, right = l2, r2
        for i in range(1, self._log + 1):
            if ((left >> i) << i) != left:
                self._update(left >> i)
            if ((right >> i) << i) != right:
                self._update((right - 1) >> i)

    def max_right(self, left, g):
        """g(op(a[left], ..., a[r - 1])) が True となる最大の r（g は単調）"""
        assert 0 <= left <= self._n
        assert g(self._e)
        if left == self._n:
            return self._n
        left += self._size
        for i in range(self._log, 0, -1):
            self._push(left >> i)
        sm = self._e
        first = True
        while first or (left & -left) != left:
            first = False
            while left % 2 == 0:
                left >>= 1
            nxt = self._op(sm, self._d[left])
            if not g(nxt):
                while left < self._size:
                    self._push(left)
                    left *= 2
                    nxt = self._op(sm, self._d[left])
                    if g(nxt):
                        sm = nxt
                        left += 1
                return left - self._size
            sm = nxt
            left += 1
        return self._n

    def min_left(self, right, g):
        """g(op(a[l], ..., a[right - 1])) が True となる最小の l（g は単調）"""
        assert 0 <= right <= self._n
        assert g(self._e)
        if right == 0:
            return 0
        right += self._size
        for i in range(self._log, 0, -1):
            self._push((right - 1) >> i)
        sm = self._e
        first = True
        while first or (right & -right) != right:
            first = False
            right -= 1
            while right > 1 and right % 2:
                right >>= 1
            nxt = self._op(self._d[right], sm)
            if not g(nxt):
                while right < self._size:
                    self._push(right)
                    right = 2 * right + 1
                    nxt = self._op(self._d[right], sm)
                    if g(nxt):
                        sm = nxt
                        right -= 1
                return right + 1 - self._size
            sm = nxt
        return 0

    def __str__(self):
        """現在の配列（保留中の更新をすべて反映した値）。O(N) なのでデバッグ用"""
        for k in range(1, self._size):
            self._push(k)
        return str(self._d[self._size:self._size + self._n])
    __repr__ = __str__
    def _update(self, k):
        self._d[k] = self._op(self._d[2 * k], self._d[2 * k + 1])
    def _all_apply(self, k, f):
        """ノード k のデータに f を作用させ、子へ渡す分をタグに溜める"""
        self._d[k] = self._mapping(f, self._d[k])
        if k < self._size:
            self._lz[k] = self._composition(f, self._lz[k])
    def _push(self, k):
        """ノード k に溜まった更新を子に下ろす"""
        self._all_apply(2 * k, self._lz[k])
        self._all_apply(2 * k + 1, self._lz[k])
        self._lz[k] = self._id

#util
def ALPHAS(small=True): return "".join([chr(i + ord("a")*small + ord("A")*(not small)) for i in range(26)])
def SIGN(x): return 1 if x > 0 else -1
# 標準入力
def DEFAULT_FMAPI(i, x):
    t = x[1:] if x[0] in "+-" else x
    if t.isdecimal() and (t == "0" or t[0] != "0"): return int(x)
    return x
def G0(): return DEFAULT_FMAPI(None, input().rstrip())
def G1(f=DEFAULT_FMAPI): return F.mapi(f, input().split())
def G2(length, f=DEFAULT_FMAPI, use_tuple=True, vert=False):
    """
    f: f(i,x), use_tuple: tuple or list, vert: 縦方向で受け取る
    """
    if length == 0: return []
    ss = [input().split() for _ in range(length)]
    size = max(map(lambda s:len(s), ss))
    def pack(xs): return tuple(xs+([None]*(size-len(xs))))
    xss = [pack(list(F.mapi(f, s))) for s in ss]
    if not use_tuple: return list(map(list, xss))
    if vert: return [[xss[i][j] for i in range(length)] for j in range(size)]
    return xss
def G2_GRID(length): 
    """文字のGRID専用"""
    return [list(input().rstrip()) for _ in range(length)]
# to_string
def TO_S(s, sep=" "):
    def ARR_TO_S(xs, f=str, sep=" "): return sep.join(map(f, xs))
    def ARRS_TO_S(xss, f=str, sep=" "): return "\n".join(map(lambda xs: ARR_TO_S(xs, f, sep), xss))
    def BOOL_TO_S(flg): return "Yes" if flg else "No"
    if isinstance(s, bool): return BOOL_TO_S(s)
    if isinstance(s, (list, tuple)):
        if len(s) == 0 or (not isinstance(s[0], (list, tuple))): return ARR_TO_S(s, sep=sep)
        else: return ARRS_TO_S(s, sep=sep)
    return f"{s}"
def PUT(ans, sep=" "): return print(TO_S(ans, sep))
def BINS(n): return format(n, "b")
#-------------------------------------------------------------------------------------------------
# main

def f_for_mapi(i,x):
    return int(x)-1

def main_solver():
    N,M = G0()
    A = G1()
    T = G2(N)

def main():
    T = 1
    # T = GN()
    for _ in range(T): main_solver()
if __name__ == "__main__": main()

