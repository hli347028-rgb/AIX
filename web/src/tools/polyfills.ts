/**
 * 钱包内置 WebView 往往比系统浏览器旧若干版本。这里只补项目首屏和业务页
 * 实际使用、且无法由 esbuild 语法降级解决的运行时 API。
 */
if (typeof Promise.allSettled !== 'function') {
  Promise.allSettled = function allSettled<T>(
    values: Iterable<T | PromiseLike<T>>,
  ): Promise<PromiseSettledResult<Awaited<T>>[]> {
    return Promise.all(
      Array.from(values, (value) =>
        Promise.resolve(value).then(
          (result) => ({ status: 'fulfilled', value: result } as PromiseFulfilledResult<Awaited<T>>),
          (reason) => ({ status: 'rejected', reason } as PromiseRejectedResult),
        ),
      ),
    )
  }
}

export {}
