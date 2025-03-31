# ID 추출 방식 수정 가이드

각 크롤러의 ID 추출 방식이 사이트마다 다르므로 아래와 같이 수정해야 합니다.

## 수정해야 할 내용

1. **fmkorea.go**:
   원본: `id, err := helpers.GetSplitPart(link, "/", 3)`
   이 방식으로 URL 경로에서 3번째 부분을 ID로 추출합니다.

2. **ppom.go** 및 **ppom_en.go**:
   원본: `id, err := helpers.GetSplitPart(link, "no=", 1)`
   URL에서 "no=" 파라미터 이후 값을 ID로 추출합니다.

3. **quasar.go**:
   원본: `id, err := helpers.GetSplitPart(link, "/", 6)`
   URL 경로에서 6번째 부분을 ID로 추출합니다.

4. **clien.go**:
   원본: `id, err := helpers.GetSplitPart(strings.Split(link, "?")[0], "/", 6)`
   쿼리 파라미터를 제거한 URL 경로에서 6번째 부분을 ID로 추출합니다.

5. **ruliweb.go**:
   원본: `id, err := helpers.GetSplitPart(strings.Split(link, "?")[0], "/", 7)`
   쿼리 파라미터를 제거한 URL 경로에서 7번째 부분을 ID로 추출합니다.

6. **arca.go**:
   원본: `id, err := helpers.GetSplitPart(strings.Split(link, "?")[0], "/", 5)`
   쿼리 파라미터를 제거한 URL 경로에서 5번째 부분을 ID로 추출합니다.

7. **coolandjoy.go**:
   원본: `id, err := helpers.GetSplitPart(link, "/", 5)`
   URL 경로에서 5번째 부분을 ID로 추출합니다.

## 해결책

factory.go 파일의 IDExtractor를 각 크롤러에 맞게 다음과 같이 수정해야 합니다:

```go
// Clien
IDExtractor: func(link string) (string, error) {
    baseLink := strings.Split(link, "?")[0]
    return helpers.GetSplitPart(baseLink, "/", 6)
},

// Ruliweb
IDExtractor: func(link string) (string, error) {
    baseLink := strings.Split(link, "?")[0]
    return helpers.GetSplitPart(baseLink, "/", 7)
},

// FMKorea
IDExtractor: func(link string) (string, error) {
    return helpers.GetSplitPart(link, "/", 3)
},

// Ppom & PpomEn
IDExtractor: func(link string) (string, error) {
    return helpers.GetSplitPart(link, "no=", 1)
},

// Quasar
IDExtractor: func(link string) (string, error) {
    return helpers.GetSplitPart(link, "/", 6)
},

// Damoang
IDExtractor: func(link string) (string, error) {
    return helpers.GetSplitPart(link, "/", 4)
},

// Arca
IDExtractor: func(link string) (string, error) {
    baseLink := strings.Split(link, "?")[0]
    return helpers.GetSplitPart(baseLink, "/", 5)
},

// Coolandjoy
IDExtractor: func(link string) (string, error) {
    return helpers.GetSplitPart(link, "/", 5)
},
```

이렇게 수정하면 각 크롤러가 원래 코드와 동일한 방식으로 ID를 추출할 수 있습니다.