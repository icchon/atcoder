# import math
from collections import defaultdict
import sys
from collections import Counter
from collections import deque
# import copy 
# import itertools
# import heapq
# import bisect
# import random
# import sympy
# import numpy as np

sys.setrecursionlimit(100000000)
# "a" + "b" <=> "".join(["a", "b"])　再帰　はCPython
input = sys.stdin.readline

class F:
    @staticmethod
    def LAMBDA(f): return lambda x:f(x)
    @staticmethod
    def hd(xs): return None if F.empty(xs) else xs[0]
    @staticmethod
    def tail(xs): return None if F.empty(xs) else xs[-1]
    @staticmethod
    def headd_tails(xs):
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
    def maplist(f, xs): return F.compose(list, F.map_curry(f))(xs)
    @staticmethod
    def inc(x): return x+1
    @staticmethod
    def dec(x): return x-1
    @staticmethod
    def compose(f, g): return lambda x:f(g(x))
    @staticmethod
    def fold_left(f, xs, x0):
        acc = x0
        for x in xs:
            acc = f(acc, x)
        return acc
    @staticmethod
    def any(xs, f=bool):
        for x in xs:
            if f(x):
                return True
        return False
    @staticmethod
    def all(xs, f=bool):
        for x in xs:
            if not f(x):
                return False
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
    def n_empty(n): return [[] for _ in range(n)]

#util
def ALPHAS(small=True): return "".join([chr(i + ord("a")*small + ord("A")*(not small)) for i in range(26)])
def SIGN(x): return 1 if x >= 0 else -1
# 標準入力
def GET_N(): return int(input().rstrip())
def GET_S(): return input().rstrip()
def GET_ARR(idx=False): return F.maplist(F.compose(F.dec, int) if idx else F.LAMBDA(int), input().split())
def GET_ARRS(length, idx=False):
    res = None
    for _ in range(length):
        lst = input().split()
        if res is None: res = [[] for _ in range(len(lst))] 
        for i, v in F.compose(enumerate, F.map_curry(F.compose(F.dec, int) if idx else F.LAMBDA(int)))(lst): res[i].append(v)
    return F.hd(res) if len(res) <= 1 else res
def GET_ARR_TUP(length, idx=False):
    res = []
    for _ in range(length):
        t = F.compose(tuple, F.map_curry(F.compose(F.dec, int) if idx else F.LAMBDA(int)))(input().split())
        res.append(F.hd(t) if len(t) <= 1 else t)
    return res
# to_string
def ARR_TO_S(xs, f=str): return " ".join(map(f, xs))
def ARRS_TO_S(xss, f=str): return "\n".join(map(lambda xs: ARR_TO_S(xs, f), xss))

#-------------------------------------------------------------------------------------------------
# main

def main():
    N = GET_N()
    


if __name__ == "__main__":
    main()
