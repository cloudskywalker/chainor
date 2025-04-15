package chainor

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPutAndGet(t *testing.T) {
	Convey("Opreate mapbucket", t, func() {
		mp := newSyncMapBucket()

		mp.put("1", "a")
		mp.put("1", "b")
		mp.put("1", "c")
		mp.put("2", "d")
		mp.put("2", "e")
		mp.put("2", "f")

		So(mp.exist("1"), ShouldBeTrue)
		So(mp.exist("2"), ShouldBeTrue)
		So(mp.exist("3"), ShouldBeFalse)

		So(mp.lenB("1"), ShouldEqual, 3)
		So(mp.lenB("2"), ShouldEqual, 3)

		varray := make([]string, 0, mp.lenB("1"))
		iter := mp.iteratorB("1")
		for iter.HasNext() {
			v := iter.Next().(string)
			varray = append(varray, v)
		}
		So(varray, ShouldResemble, []string{"a", "b", "c"})

		vmap := make(map[string]bool)
		mp.eachKey(func(bkey any) {
			vmap[bkey.(string)] = true
		})
		So(vmap, ShouldResemble, map[string]bool{"1": true, "2": true})
	})
}
