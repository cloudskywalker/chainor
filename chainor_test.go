package chainor

import (
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/huandu/go-clone"
	. "github.com/smartystreets/goconvey/convey"
	//"go.uber.org/goleak"
)

//func TestMain(m *testing.M) {
//	goleak.VerifyTestMain(m)
//}

func TestNext(t *testing.T) {
	Convey("Next and invoke", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)
		param := map[string]string{"1": "a"}

		chn := NewChainor(WithChainorName("test"), WithChainorParam(param), WithChainorProps(M{"2": "b"})).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult, ShouldBeNil)
				c.So(ctx.ChainorName(), ShouldEqual, "test")
				c.So(ctx.ChainorParam(), ShouldEqual, param)
				c.So(ctx.ChainorProps(), ShouldResemble, M{"2": "b"})
				c.So(ctx.TaskParam(), ShouldEqual, 5)
				return 2, nil
			}, WithTaskParam(5)).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult[0], ShouldEqual, 2)
				c.So(ctx.TaskProps(), ShouldResemble, M{"2": 1})
				c.So(ctx.Param(), ShouldEqual, 78)
				c.So(ctx.Props(), ShouldResemble, M{"rt": "op"})
				return 3, nil
			}, WithTaskProps(M{"2": 1}))

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 3)
			wg.Done()
		}, nil, WithParam(78), WithProps(M{"rt": "op"}))

		wg.Wait()
	})

	Convey("Next with parallel", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor().
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 89, nil
			}).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult, ShouldResemble, []any{89})
				c.So(ctx.TaskParam(), ShouldEqual, 5)
				return 2, nil
			}, WithTaskParam(5),
				WithParallelFunc(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					c.So(lastResult, ShouldResemble, []any{89})
					return 7, nil
				}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
					c.So(lastResult, ShouldResemble, []any{89})
					return 8, nil
				}),
				WithParallel(2)).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(cloneAndSortResult(lastResult), ShouldResemble, []any{2, 2, 7, 7, 8, 8})
				return 3, nil
			})

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 3)
			wg.Done()
		}, nil)

		wg.Wait()
	})

	Convey("Next with workerpool", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor().
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				ctx.WithValue("io", 0)
				return 2, nil
			}, WithTaskParam(5),
				WithParallelFunc(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return 7, nil
				}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return 8, nil
				}),
				WithWorkerPool(5)).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(cloneAndSortResult(lastResult), ShouldResemble, []any{2, 7, 8})
				c.So(ctx.Value("io").(int), ShouldEqual, 0)
				return 3, nil
			})

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 3)
			wg.Done()
		}, nil)

		wg.Wait()
	})

	Convey("Next with any pass", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor().
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 2, nil
			}, WithParallelFunc(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 7, nil
			}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 8, nil
			}), WithWorkerPool(5), WithAnyPassed(func(result any) bool {
				if result == 7 {
					return true
				}
				return false
			}))

		Invoke(chn, func(result []any) {
			c.So(result, ShouldResemble, []any{7})
			wg.Done()
		}, nil)

		wg.Wait()

		Convey("Next with no pass", func(c C) {
			wg = sync.WaitGroup{}
			wg.Add(1)

			chn = NewChainor().
				Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return 2, nil
				}, WithParallelFunc(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return 7, nil
				}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return 8, nil
				}), WithWorkerPool(5), WithAnyPassed(func(result any) bool {
					if result == 9 {
						return true
					}
					return false
				}))

			Invoke(chn, nil, func(err error) {
				c.So(err, ShouldEqual, ErrNoPassed)
				wg.Done()
			})

			wg.Wait()
		})
	})

	Convey("Next with skip", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor().
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 5, nil
			}).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 6, nil
			}, WithSkipped(func(result []any) bool {
				if result[0] == 5 {
					return true
				}
				return false
			}))

		Invoke(chn, func(result []any) {
			c.So(result, ShouldResemble, []any{5})
			wg.Done()
		}, nil)

		wg.Wait()
	})

	Convey("Next with error", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor().
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 5, nil
			}).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 6, errors.New("test error")
			}, WithSkipped(func(result []any) bool {
				if result[0] == 9 {
					return true
				}
				return false
			}))

		Invoke(chn, nil, func(err error) {
			c.So(err.Error(), ShouldEqual, "test error")
			wg.Done()
		})

		wg.Wait()

		Convey("Next with timeout", func(c C) {
			wg = sync.WaitGroup{}
			wg.Add(1)

			chn = NewChainor().
				Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return 5, nil
				}).
				Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					time.Sleep(time.Second)
					return 8, nil
				})

			Invoke(chn, func(result []any) {
				c.So(result, ShouldResemble, []any{8})
				wg.Done()
			}, func(err error) {
				c.So(err, ShouldEqual, ErrTimeout)
				wg.Done()
			}, WithTimeout(time.Second))

			wg.Wait()
		})
	})
}

func cloneAndSortResult(r []any) []any {
	cr := clone.Clone(r).([]any)
	sort.SliceStable(cr, func(i, j int) bool {
		return cr[i].(int) < cr[j].(int)
	})
	return cr
}

func TestNextN(t *testing.T) {
	Convey("NextN and invoke", t, func(c C) {
		param := map[string]string{"1": "a"}

		Register("pipe1", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(ctx.TaskParam(), ShouldEqual, 5)
			c.So(ctx.ChainorParam(), ShouldResemble, param)
			c.So(lastResult, ShouldResemble, []any{9})
			return 1, nil
		}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{9})
			return 2, nil
		})
		Register("pipe1", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{9})
			return 3, nil
		})

		Register("pipe2", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10, 10})
			return 31, nil
		})

		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor(WithChainorName("test1"), WithChainorParam(param)).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 9, nil
			}).
			NextN("pipe1", WithTaskParam(5)).
			Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				// 直接 sort 会修改 lastResult，触发竞争检测错误
				c.So(cloneAndSortResult(lastResult), ShouldResemble, []any{1, 2, 3})
				return 10, nil
			}, WithParallel(2)).
			NextN("pipe2", WithAnyPassed(func(result any) bool {
				c.So(result, ShouldResemble, 31)
				return true
			}))

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 31)
			wg.Done()
		}, nil)

		wg.Wait()
	})
}

func TestSwitch2(t *testing.T) {
	Convey("Switch2", t, func(c C) {
		Register("f1", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{9})
			return 1, nil
		}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{9})
			return 2, nil
		})
		Register("f1", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{9})
			return 3, nil
		})

		Register("f2", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10})
			return 21, nil
		})

		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor(WithChainorName("test2"))
		chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			ctx.WithValue("uu", "oo")
			return 9, nil
		}).Switch2(func(lastResult []any) (result any) {
			return 7
		}).Case(7, func(chn1 *Chainor) {
			chn1.NextN("f1", WithTaskParam(5)).
				Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					c.So(cloneAndSortResult(lastResult), ShouldResemble, []any{1, 2, 3})
					return 10, nil
				}).
				NextN("f2", WithAnyPassed(func(result any) bool {
					c.So(result, ShouldResemble, 21)
					return true
				})).
				Switch2(func(lastResult []any) (result any) {
					return lastResult[0]
				}).
				Case(20, func(c2 *Chainor) {

				}).
				Default(func(c2 *Chainor) {
					c2.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
						c.So(lastResult, ShouldResemble, []any{21})
						c.So(ctx.ChainorName(), ShouldEqual, "test2")
						c.So(ctx.Value("uu"), ShouldEqual, "oo")
						c.So(ctx.Param(), ShouldEqual, 69)
						return 88, nil
					})
				})
		}).Case(12, func(_ *Chainor) {
		}).Default(func(_ *Chainor) {
		})

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 88)
			wg.Done()
		}, nil, WithParam(69))

		wg.Wait()
	})

	Convey("Switch2 exception", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor()
		chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			return 99, nil
		}).Switch2(func(lastResult []any) (result any) {
			return 7
		}).Default(func(_ *Chainor) {
		})

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 99)
			wg.Done()
		}, nil)

		wg.Wait()

		Convey("Switch2 with next error", func(c C) {
			wg = sync.WaitGroup{}
			wg.Add(1)

			chn = NewChainor()
			chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 99, nil
			}).Switch2(func(lastResult []any) (result any) {
				return 7
			}).Default(func(cx *Chainor) {
				cx.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					c.So(lastResult, ShouldResemble, []any{99})
					return nil, errors.New("test condition error")
				})
			})

			Invoke(chn, nil, func(err error) {
				c.So(err.Error(), ShouldEqual, "test condition error")
				wg.Done()
			})

			wg.Wait()

			Convey("Switch2 with error", func(c C) {
				wg = sync.WaitGroup{}
				wg.Add(1)

				chn = NewChainor()
				chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return nil, errors.New("bb")
				}).Switch2(func(lastResult []any) (result any) {
					return 7
				}).Default(func(cx *Chainor) {
					cx.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
						c.So(lastResult, ShouldResemble, []any{99})
						return nil, errors.New("test condition error")
					})
				})

				Invoke(chn, nil, func(err error) {
					c.So(err.Error(), ShouldEqual, "bb")
					wg.Done()
				})

				wg.Wait()

				Convey("Switch2 with timeout error", func(c C) {
					wg = sync.WaitGroup{}
					wg.Add(1)

					chn = NewChainor()
					chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
						return 10, nil
					}).Switch2(func(lastResult []any) (result any) {
						return 7
					}).Case(8, func(c *Chainor) {

					}).Default(func(cx *Chainor) {
						cx.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
							time.Sleep(time.Second)
							return 88, nil
						}, WithParallelFunc(func(ctx *TaskContext, lastResult []any) (result any, err error) {
							return 99, nil
						}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
							return 100, nil
						}))
					})

					Invoke(chn, func(result []any) {
						c.So(cloneAndSortResult(result), ShouldResemble, []any{88, 99, 100})
						wg.Done()
					}, func(err error) {
						c.So(err, ShouldEqual, ErrTimeout)
						wg.Done()
					}, WithTimeout(time.Second))

					wg.Wait()

					Convey("Switch2 parallel partition timeout", func(c C) {
						wg = sync.WaitGroup{}
						wg.Add(1)

						chn = NewChainor()
						chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
							return 10, nil
						}).Switch2(func(lastResult []any) (result any) {
							return 7
						}).Case(8, func(c *Chainor) {

						}).Default(func(cx *Chainor) {
							cx.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
								time.Sleep(time.Second)
								return 88, nil
							}, WithParallelFunc(func(ctx *TaskContext, lastResult []any) (result any, err error) {
								time.Sleep(2 * time.Second)
								return 99, nil
							}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
								return 100, nil
							}))
						})

						Invoke(chn, nil, func(err error) {
							c.So(err, ShouldEqual, ErrTimeout)
							wg.Done()
						}, WithTimeout(time.Second))

						wg.Wait()

						Convey("Switch2 any passed partition timeout", func(c C) {
							wg = sync.WaitGroup{}
							wg.Add(1)

							chn = NewChainor()
							chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
								return 10, nil
							}).Switch2(func(lastResult []any) (result any) {
								return 7
							}).Case(8, func(c *Chainor) {

							}).Default(func(cx *Chainor) {
								cx.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
									time.Sleep(time.Second)
									return 88, nil
								}, WithParallelFunc(func(ctx *TaskContext, lastResult []any) (result any, err error) {
									time.Sleep(2 * time.Second)
									return 99, nil
								}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
									return 100, nil
								}), WithAnyPassed())
							})

							Invoke(chn, func(result []any) {
								c.So(result[0], ShouldEqual, 100)
								wg.Done()
							}, nil, WithTimeout(time.Second))

							wg.Wait()
						})
					})
				})
			})
		})
	})
}

func TestNextNWithMore(t *testing.T) {
	Convey("NextN with more", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		Register("demo1", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10})
			return 1, nil
		}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10})
			return 2, nil
		})

		chn := NewChainor()
		chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			return 10, nil
		}).Switch2(func(lastResult []any) (result any) {
			return 7
		}).Default(func(cx *Chainor) {
			cx.NextN("demo1", WithSkipped(func(result []any) bool {
				if result[0] == 9 {
					return true
				}
				return false
			}), WithWorkerPool(5), WithAnyPassed(func(result any) bool {
				if result == 2 {
					return true
				}
				return false
			}))
			cx.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult, ShouldResemble, []any{2})
				return 88, nil
			})
		})

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 88)
			wg.Done()
		}, nil)

		wg.Wait()
	})
}

func TestSwitch(t *testing.T) {
	Convey("Switch with next", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		Register("s1", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10})
			return 1, nil
		}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10})
			return 2, nil
		})

		chn := NewChainor()
		chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			ctx.WithValue("1", "2")
			return 10, nil
		}).Switch(func(lastResult []any) (result any) {
			return 7
		}).Case(8, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(ctx.TaskParam(), ShouldEqual, 90)
			return 15, nil
		}, WithTaskParam(90),
		).CaseN(7, "s1", WithTaskParam(100), WithSkipped(func(result []any) bool {
			// 测试是否会忽略
			return true
		})).Default(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult[0], ShouldEqual, 10)
			c.So(ctx.Value("1"), ShouldEqual, "2")
			return 68, nil
		}).Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(cloneAndSortResult(lastResult), ShouldResemble, []any{1, 2})
			c.So(ctx.TaskParam(), ShouldEqual, 99)
			c.So(ctx.Value("1"), ShouldEqual, "2")
			return 166, nil
		}, WithTaskParam(99))

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 166)
			wg.Done()
		}, nil, WithParam(9))

		wg.Wait()
	})

	Convey("Switch with more", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		Register("s2", func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10})
			return 1, nil
		}, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult, ShouldResemble, []any{10})
			return 2, nil
		})

		chn := NewChainor()
		chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			ctx.WithValue("1", "2")
			return 10, nil
		}).Switch(func(lastResult []any) (result any) {
			return 7
		}).Case(8, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(ctx.TaskParam(), ShouldEqual, 90)
			return 15, nil
		}, WithTaskParam(90),
		).DefaultN("s2", WithParallel(2)).Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(cloneAndSortResult(lastResult), ShouldResemble, []any{1, 1, 2, 2})
			c.So(ctx.TaskParam(), ShouldEqual, 99)
			c.So(ctx.Value("1"), ShouldEqual, "2")
			return 166, nil
		}, WithTaskParam(99))

		Invoke(chn, func(result []any) {
			c.So(result[0], ShouldEqual, 166)
			wg.Done()
		}, nil, WithParam(9))

		wg.Wait()
	})

	Convey("Switch exception", t, func(c C) {
		wg := sync.WaitGroup{}
		wg.Add(1)

		chn := NewChainor()
		chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			return 10, nil
		}).Switch(func(lastResult []any) (result any) {
			return 7
		}).Case(8, func(ctx *TaskContext, lastResult []any) (result any, err error) {
			return 15, nil
		}).Default(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			c.So(lastResult[0], ShouldEqual, 10)
			return nil, errors.New("default err")
		}).Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
			return nil, errors.New("next err")
		})

		Invoke(chn, func(result []any) {
		}, func(err error) {
			c.So(err.Error(), ShouldEqual, "default err")
			wg.Done()
		}, WithParam(9))

		wg.Wait()

		Convey("Switch exception more", func(c C) {
			wg = sync.WaitGroup{}
			wg.Add(2)

			Register("s3", func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult, ShouldResemble, []any{10})
				return 1, nil
			})
			Register("s3", func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult, ShouldResemble, []any{10})
				return 2, errors.New("nextn err")
			})

			chn = NewChainor()
			chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 10, nil
			}).Switch(func(lastResult []any) (result any) {
				return 7
			}).CaseN(7, "s3", WithTaskParam(100), WithSkipped(func(result []any) bool {
				// 测试是否会忽略
				return true
			})).Default(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult[0], ShouldEqual, 15)
				return nil, errors.New("default err")
			}).Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return nil, errors.New("next err")
			})

			Invoke(chn, func(result []any) {
			}, func(err error) {
				c.So(err.Error(), ShouldEqual, "nextn err")
				wg.Done()
			}, WithParam(9))

			chn1 := NewChainor()
			chn1.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return 10, nil
			}).Switch(func(lastResult []any) (result any) {
				return 7
			}).Default(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				c.So(lastResult[0], ShouldEqual, 10)
				return 12, nil
			}).Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
				return nil, errors.New("next err")
			})

			Invoke(chn1, func(result []any) {
			}, func(err error) {
				c.So(err.Error(), ShouldEqual, "next err")
				wg.Done()
			}, WithParam(9))

			wg.Wait()

			Convey("Switch timeout", func(c C) {
				wg = sync.WaitGroup{}
				wg.Add(1)

				chn = NewChainor()
				chn.Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					return 10, nil
				}).Switch(func(lastResult []any) (result any) {
					return 7
				}).Default(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					c.So(lastResult[0], ShouldEqual, 10)
					return 12, nil
				}).Next(func(ctx *TaskContext, lastResult []any) (result any, err error) {
					time.Sleep(2 * time.Second)
					return 15, nil
				})

				Invoke(chn, func(result []any) {
				}, func(err error) {
					c.So(err, ShouldEqual, ErrTimeout)
					wg.Done()
				}, WithTimeout(time.Second))

				wg.Wait()
			})
		})
	})

}
