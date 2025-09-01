package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// CardInfo 存储信用卡信息
type CardInfo struct {
	Number     string
	ExpiryDate string // 格式: MM|YY
	CVV        string
}

// validateBin 验证BIN码（6-12位数字）
func validateBin(bin string) error {
	if len(bin) < 6 || len(bin) > 12 {
		return fmt.Errorf("BIN must be 6-12 digits")
	}
	if _, err := strconv.ParseInt(bin, 10, 64); err != nil {
		return fmt.Errorf("BIN must contain only digits")
	}
	return nil
}

// getCVVLength 根据BIN判断CVV长度（Amex为4位，其他为3位）
func getCVVLength(bin string) int {
	if strings.HasPrefix(bin, "34") || strings.HasPrefix(bin, "37") {
		return 4
	}
	return 3
}

// generateCheckDigit 使用Luhn算法生成校验位
func generateCheckDigit(number string) string {
	sum := 0
	shouldDouble := true
	for i := len(number) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(number[i]))
		if shouldDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		shouldDouble = !shouldDouble
	}
	return strconv.Itoa((10 - (sum % 10)) % 10)
}

// generateRandomCVV 生成随机CVV
func generateRandomCVV(length int) string {
	max := int(1e9) // 足够大的范围
	cvv := rand.Intn(max)
	return fmt.Sprintf("%0*d", length, cvv)[:length]
}

// generateRandomExpiryDate 生成随机有效期（当前年份后的1-5年）
func generateRandomExpiryDate() string {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())
	year := currentYear + rand.Intn(5) + 1
	var month int
	if year == currentYear {
		month = currentMonth + rand.Intn(13-currentMonth)
	} else {
		month = rand.Intn(12) + 1
	}
	return fmt.Sprintf("%02d|%02d", month, year%100)
}

// generateCard 生成单个信用卡信息
func generateCard(bin string, quantity int, fixedExpiryDate, fixedCVV string) ([]CardInfo, error) {
	// 验证BIN
	if err := validateBin(bin); err != nil {
		return nil, err
	}

	// 确定CVV长度
	cvvLength := getCVVLength(bin)
	targetLength := 16
	if cvvLength == 4 {
		targetLength = 15 // Amex卡号长度为15
	}

	// 验证固定CVV
	if fixedCVV != "" {
		if len(fixedCVV) != cvvLength || !isNumeric(fixedCVV) {
			return nil, fmt.Errorf("CVV must be %d digits", cvvLength)
		}
	}

	// 验证固定有效期
	if fixedExpiryDate != "" {
		if !isValidExpiryDate(fixedExpiryDate) {
			return nil, fmt.Errorf("invalid expiry date, use MM|YY format")
		}
	}

	// 生成多张卡
	var cards []CardInfo
	for i := 0; i < quantity; i++ {
		// 生成卡号
		number := bin
		for len(number) < targetLength-1 {
			number += strconv.Itoa(rand.Intn(10))
		}
		number += generateCheckDigit(number)

		// 生成或使用固定的CVV和有效期
		cvv := fixedCVV
		if cvv == "" {
			cvv = generateRandomCVV(cvvLength)
		}
		expiry := fixedExpiryDate
		if expiry == "" {
			expiry = generateRandomExpiryDate()
		}

		cards = append(cards, CardInfo{
			Number:     number,
			ExpiryDate: expiry,
			CVV:        cvv,
		})
	}
	return cards, nil
}

// isNumeric 检查字符串是否为纯数字
func isNumeric(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

// isValidExpiryDate 验证有效期格式（MM|YY）
func isValidExpiryDate(date string) bool {
	parts := strings.Split(date, "|")
	if len(parts) != 2 {
		return false
	}
	month, err := strconv.Atoi(parts[0])
	if err != nil || month < 1 || month > 12 {
		return false
	}
	year, err := strconv.Atoi(parts[1])
	if err != nil || year < 0 || year > 99 {
		return false
	}
	return true
}

func main() {
	// 设置随机种子
	rand.New(rand.NewSource(time.Now().UnixNano()))
	// 示例：生成10张卡，BIN为"424242"，无固定有效期和CVV
	cards, err := generateCard("423568", 10, "", "")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// 打印生成的卡信息
	for i, card := range cards {
		fmt.Printf("Card %d: %s | %s | %s\n", i+1, card.Number, card.ExpiryDate, card.CVV)
	}
}
