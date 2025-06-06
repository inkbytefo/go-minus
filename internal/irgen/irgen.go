package irgen

import (
	"fmt"
	"strings"

	"github.com/inkbytefo/go-minus/internal/ast"
	"github.com/inkbytefo/go-minus/internal/semantic"
	"github.com/inkbytefo/go-minus/internal/token"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// IRGenerator converts AST to LLVM IR or similar intermediate code.
type IRGenerator struct {
	errors         []string
	moduleName     string
	module         *ir.Module
	currentFunc    *ir.Func
	currentBB      *ir.Block
	symbolTable    map[string]value.Value   // Symbol table
	typeTable      map[string]types.Type    // Type table
	classTable     map[string]*ClassInfo    // Class table
	templateTable  map[string]*TemplateInfo // Template table
	exceptionStack []*ExceptionInfo         // Exception stack
	analyzer       *semantic.Analyzer       // Semantic analyzer
	debugInfo      *DebugInfo               // Debug information
	generateDebug  bool                     // Generate debug information?
	sourceFile     string                   // Source file name
	sourceDir      string                   // Source file directory
	labelCounter   int                      // Counter for unique labels
}

// New creates a new IRGenerator.
func New() *IRGenerator {
	module := ir.NewModule()
	return &IRGenerator{
		errors:         []string{},
		module:         module,
		moduleName:     "gominus_module",
		symbolTable:    make(map[string]value.Value),
		typeTable:      make(map[string]types.Type),
		classTable:     make(map[string]*ClassInfo),
		templateTable:  make(map[string]*TemplateInfo),
		exceptionStack: make([]*ExceptionInfo, 0),
		generateDebug:  false,
		sourceFile:     "",
		sourceDir:      "",
	}
}

// NewWithAnalyzer creates a new IRGenerator with a semantic analyzer.
func NewWithAnalyzer(analyzer *semantic.Analyzer) *IRGenerator {
	module := ir.NewModule()
	return &IRGenerator{
		errors:         []string{},
		module:         module,
		moduleName:     "gominus_module",
		symbolTable:    make(map[string]value.Value),
		typeTable:      make(map[string]types.Type),
		classTable:     make(map[string]*ClassInfo),
		templateTable:  make(map[string]*TemplateInfo),
		exceptionStack: make([]*ExceptionInfo, 0),
		analyzer:       analyzer,
		generateDebug:  false,
		sourceFile:     "",
		sourceDir:      "",
	}
}

// SetSourceFile sets the source file and directory for debug information.
func (g *IRGenerator) SetSourceFile(filename, directory string) {
	g.sourceFile = filename
	g.sourceDir = directory
}

// EnableDebugInfo enables or disables debug information generation.
func (g *IRGenerator) EnableDebugInfo(enable bool) {
	g.generateDebug = enable
}

// Errors, IR üretimi sırasında karşılaşılan hataları döndürür.
func (g *IRGenerator) Errors() []string {
	return g.errors
}

// ReportError, bir hata mesajı ekler.
func (g *IRGenerator) ReportError(format string, args ...any) {
	g.errors = append(g.errors, fmt.Sprintf(format, args...))
}

// InitDebugInfo initializes debug information with source file and directory.
func (g *IRGenerator) InitDebugInfo(sourceFile, sourceDir string) {
	g.generateDebug = true
	g.sourceFile = sourceFile
	g.sourceDir = sourceDir
	g.debugInfo = NewDebugInfo(g.module)
	g.debugInfo.InitCompileUnit(sourceFile, sourceDir, "GO-Minus Compiler", false, "", 0)
}

// GenerateProgram, programın AST'sinden IR üretir.
func (g *IRGenerator) GenerateProgram(program *ast.Program) (string, error) {
	// Modülü sıfırla
	g.module = ir.NewModule()
	g.module.SourceFilename = g.moduleName

	// Temel tipleri tanımla
	g.defineBasicTypes()

	// Hata ayıklama bilgisi üretimini başlat
	if g.generateDebug {
		g.debugInfo = NewDebugInfo(g.module)
		g.debugInfo.InitCompileUnit(g.sourceFile, g.sourceDir, "GO-Minus Compiler", false, "", 0)
	}

	// Hata kontrolü
	if len(g.Errors()) > 0 {
		return "", fmt.Errorf("IR üretimi sırasında hatalar oluştu: %v", g.Errors())
	}

	// AST düğümlerini gezerek IR üretme
	for _, stmt := range program.Statements {
		// Hata ayıklama bilgisi için konum bilgisini ayarla
		if g.generateDebug && stmt.Pos().IsValid() {
			pos := stmt.Pos()
			g.debugInfo.SetLocation(pos.Line, pos.Column, g.sourceFile)
		}

		g.generateStatement(stmt)
	}

	// Main fonksiyonu yoksa oluştur
	if g.getFunction("main") == nil {
		g.createMainFunction()
	}

	// Hata kontrolü
	if len(g.Errors()) > 0 {
		return "", fmt.Errorf("IR üretimi sırasında hatalar oluştu: %v", g.Errors())
	}

	// Optimizasyon geçişleri uygula
	g.applyOptimizations()

	// Modülü string olarak döndür
	return g.module.String(), nil
}

// generateImportStatement, bir import deyimi için IR üretir.
func (g *IRGenerator) generateImportStatement(stmt *ast.ImportStatement) {
	// Import statement'ları için özel bir işlem yapmıyoruz
	// Standard library binding semantic analysis'te yapılıyor
	// Bu fonksiyon sadece hata vermemek için var
}

// applyOptimizations, IR koduna optimizasyon geçişleri uygular.
func (g *IRGenerator) applyOptimizations() {
	// Şu anda optimizasyon işlemleri optimizer paketi tarafından yapılıyor
	// Bu metod, ileride doğrudan IR üzerinde optimizasyon yapmak için kullanılabilir
}

// defineBasicTypes, temel tipleri tanımlar.
func (g *IRGenerator) defineBasicTypes() {
	// Temel tipleri tanımla
	g.typeTable["int"] = types.I32
	g.typeTable["int8"] = types.I8
	g.typeTable["int16"] = types.I16
	g.typeTable["int32"] = types.I32
	g.typeTable["int64"] = types.I64
	g.typeTable["uint"] = types.I32
	g.typeTable["uint8"] = types.I8
	g.typeTable["uint16"] = types.I16
	g.typeTable["uint32"] = types.I32
	g.typeTable["uint64"] = types.I64
	g.typeTable["float"] = types.Double // Varsayılan float tipi (float64 olarak)
	g.typeTable["float32"] = types.Float
	g.typeTable["float64"] = types.Double
	g.typeTable["bool"] = types.I1
	g.typeTable["byte"] = types.I8
	g.typeTable["rune"] = types.I32
	g.typeTable["string"] = types.NewPointer(types.I8) // Basitleştirilmiş string temsili
}

// getTypeTableKeys, debug için typeTable'daki anahtarları döndürür.
func (g *IRGenerator) getTypeTableKeys() []string {
	keys := make([]string, 0, len(g.typeTable))
	for k := range g.typeTable {
		keys = append(keys, k)
	}
	return keys
}

// getFunction, belirtilen isimde bir fonksiyonu döndürür.
func (g *IRGenerator) getFunction(name string) *ir.Func {
	for _, f := range g.module.Funcs {
		if f.Name() == name {
			return f
		}
	}
	return nil
}

// createMainFunction, main fonksiyonunu oluşturur.
func (g *IRGenerator) createMainFunction() {
	// Main fonksiyonu oluştur
	mainFunc := g.module.NewFunc("main", types.I32)
	entryBlock := mainFunc.NewBlock("entry")

	// Return 0
	entryBlock.NewRet(constant.NewInt(types.I32, 0))
}

// generateStatement, bir deyim için IR üretir.
func (g *IRGenerator) generateStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.PackageStatement:
		// Paket bildirimi için özel bir işlem yapmıyoruz
		// Sadece modül adını ayarlıyoruz
		g.moduleName = s.Name.Value
	case *ast.ImportStatement:
		// Import statement'ları için özel bir işlem yapmıyoruz
		// Standard library binding semantic analysis'te yapılıyor
		g.generateImportStatement(s)
	case *ast.ExpressionStatement:
		g.generateExpression(s.Expression)
	case *ast.VarStatement:
		g.generateVarStatement(s)
	case *ast.ReturnStatement:
		g.generateReturnStatement(s)
	case *ast.BlockStatement:
		g.generateBlockStatement(s)
	case *ast.WhileStatement:
		g.generateWhileStatement(s)
	case *ast.ForStatement:
		g.generateForStatement(s)
	case *ast.SwitchStatement:
		g.generateSwitchStatement(s)
	case *ast.FunctionStatement:
		g.generateFunctionStatement(s)
	case *ast.ClassStatement:
		g.generateClassStatement(s)
	case *ast.TemplateStatement:
		g.generateTemplateStatement(s)
	case *ast.TryCatchStatement:
		g.generateTryCatchStatement(s)
	case *ast.ThrowStatement:
		g.generateThrowStatement(s)
	default:
		g.ReportError("Desteklenmeyen deyim türü: %T", s)
	}
}

// generateExpression, bir ifade için IR üretir ve değeri döndürür.
func (g *IRGenerator) generateExpression(expr ast.Expression) value.Value {
	switch e := expr.(type) {
	case *ast.Identifier:
		return g.generateIdentifier(e)
	case *ast.IntegerLiteral:
		return g.generateIntegerLiteral(e)
	case *ast.FloatLiteral:
		return g.generateFloatLiteral(e)
	case *ast.StringLiteral:
		return g.generateStringLiteral(e)
	case *ast.BooleanLiteral:
		return g.generateBooleanLiteral(e)
	case *ast.PrefixExpression:
		return g.generatePrefixExpression(e)
	case *ast.InfixExpression:
		return g.generateInfixExpression(e)
	case *ast.PostfixExpression:
		return g.generatePostfixExpression(e)
	case *ast.CallExpression:
		return g.generateCallExpression(e)
	case *ast.FunctionLiteral:
		return g.generateFunctionLiteral(e)
	case *ast.IfExpression:
		return g.generateIfExpression(e)
	case *ast.NewExpression:
		return g.generateNewExpression(e)
	case *ast.MemberExpression:
		return g.generateMemberExpression(e)
	case *ast.TemplateExpression:
		return g.generateTemplateExpression(e)
	case *ast.TryExpression:
		return g.generateTryExpression(e)
	case *ast.ArrayLiteral:
		return g.generateArrayLiteral(e)
	case *ast.IndexExpression:
		return g.generateIndexExpression(e)
	default:
		g.ReportError("Desteklenmeyen ifade türü: %T", e)
		return nil
	}
}

// getExpressionType, bir ifadenin tipini döndürür.
func (g *IRGenerator) getExpressionType(expr ast.Expression) types.Type {
	switch e := expr.(type) {
	case *ast.Identifier:
		// Tanımlayıcının tipini bul
		if val, exists := g.symbolTable[e.Value]; exists {
			return g.getValueType(val)
		}
		return nil
	case *ast.IntegerLiteral:
		return types.I32 // Varsayılan olarak int32
	case *ast.FloatLiteral:
		return types.Double // Varsayılan olarak float64
	case *ast.StringLiteral:
		return types.NewPointer(types.I8) // Basitleştirilmiş string temsili
	case *ast.BooleanLiteral:
		return types.I1
	case *ast.PrefixExpression:
		return g.getExpressionType(e.Right)
	case *ast.InfixExpression:
		// Aritmetik operatörler için
		if e.Operator == "+" || e.Operator == "-" || e.Operator == "*" || e.Operator == "/" {
			leftType := g.getExpressionType(e.Left)
			rightType := g.getExpressionType(e.Right)
			// Tip yükseltme (type promotion)
			if leftType == types.Double || rightType == types.Double {
				return types.Double
			}
			return types.I32
		}
		// Karşılaştırma operatörleri için
		if e.Operator == "==" || e.Operator == "!=" || e.Operator == "<" || e.Operator == ">" || e.Operator == "<=" || e.Operator == ">=" {
			return types.I1
		}
		return types.I32
	case *ast.ArrayLiteral:
		// Array literal için tip belirleme
		if len(e.Elements) == 0 {
			return types.NewPointer(types.I32) // Boş array
		}
		// İlk element'in tipini al
		elementType := g.getExpressionType(e.Elements[0])
		if elementType == nil {
			elementType = types.I32
		}
		return types.NewPointer(types.NewArray(uint64(len(e.Elements)), elementType))
	case *ast.IndexExpression:
		// Index expression için element tipini döndür
		arrayType := g.getExpressionType(e.Left)
		if ptrType, ok := arrayType.(*types.PointerType); ok {
			if arrType, ok := ptrType.ElemType.(*types.ArrayType); ok {
				return arrType.ElemType
			}
			return ptrType.ElemType
		}
		return types.I32
	default:
		g.ReportError("Desteklenmeyen ifade türü (tip belirlenemiyor): %T", e)
		return nil
	}
}

// getValueType, bir değerin tipini döndürür.
func (g *IRGenerator) getValueType(val value.Value) types.Type {
	if val == nil {
		return nil
	}
	return val.Type()
}

// generateConstantExpression, sabit bir ifade için IR üretir.
func (g *IRGenerator) generateConstantExpression(expr ast.Expression) constant.Constant {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return constant.NewInt(types.I32, e.Value)
	case *ast.FloatLiteral:
		return constant.NewFloat(types.Double, e.Value)
	case *ast.BooleanLiteral:
		if e.Value {
			return constant.NewInt(types.I1, 1)
		}
		return constant.NewInt(types.I1, 0)
	case *ast.StringLiteral:
		// String sabitleri için global değişken oluştur
		strConst := g.module.NewGlobalDef("", constant.NewCharArrayFromString(e.Value+"\x00"))
		return constant.NewGetElementPtr(strConst.ContentType, strConst, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
	default:
		g.ReportError("Desteklenmeyen sabit ifade türü: %T", e)
		return nil
	}
}

// Temel ifade türleri için IR üretme fonksiyonları

func (g *IRGenerator) generateIdentifier(ident *ast.Identifier) value.Value {
	// Tanımlayıcının değerini sembol tablosundan bul
	if val, exists := g.symbolTable[ident.Value]; exists {
		// Eğer değer bir pointer ise (örn. alloca), yükle
		if ptr, ok := val.(value.Value); ok && types.IsPointer(ptr.Type()) {
			if g.currentBB != nil {
				return g.currentBB.NewLoad(ptr.Type().(*types.PointerType).ElemType, ptr)
			}
		}
		return val
	}

	g.ReportError("Tanımlanmamış tanımlayıcı: %s", ident.Value)
	return nil
}

func (g *IRGenerator) generateIntegerLiteral(lit *ast.IntegerLiteral) value.Value {
	return constant.NewInt(types.I32, lit.Value)
}

func (g *IRGenerator) generateFloatLiteral(lit *ast.FloatLiteral) value.Value {
	return constant.NewFloat(types.Double, lit.Value)
}

func (g *IRGenerator) generateStringLiteral(lit *ast.StringLiteral) value.Value {
	// String sabitleri için global değişken oluştur
	strConst := g.module.NewGlobalDef("", constant.NewCharArrayFromString(lit.Value+"\x00"))
	return constant.NewGetElementPtr(strConst.ContentType, strConst, constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
}

func (g *IRGenerator) generateBooleanLiteral(lit *ast.BooleanLiteral) value.Value {
	if lit.Value {
		return constant.NewInt(types.I1, 1)
	}
	return constant.NewInt(types.I1, 0)
}

// Karmaşık ifade türleri için IR üretme fonksiyonları

func (g *IRGenerator) generatePrefixExpression(expr *ast.PrefixExpression) value.Value {
	right := g.generateExpression(expr.Right)
	if right == nil {
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, önek ifadesi değerlendirilemiyor")
		return nil
	}

	switch expr.Operator {
	case "!":
		// Boolean değil
		return g.currentBB.NewXor(right, constant.NewInt(types.I1, 1))
	case "-":
		// Sayısal negatif
		if types.IsInt(right.Type()) {
			return g.currentBB.NewSub(constant.NewInt(types.I32, 0), right)
		} else if types.IsFloat(right.Type()) {
			return g.currentBB.NewFSub(constant.NewFloat(types.Double, 0), right)
		}
	}

	g.ReportError("Desteklenmeyen önek operatörü: %s", expr.Operator)
	return nil
}

func (g *IRGenerator) generatePostfixExpression(expr *ast.PostfixExpression) value.Value {
	// Sol ifadeyi değerlendir (değişken olmalı)
	leftIdent, ok := expr.Left.(*ast.Identifier)
	if !ok {
		g.ReportError("Postfix operatörü sadece değişkenler üzerinde kullanılabilir")
		return nil
	}

	// Değişkenin adresini bul
	varName := leftIdent.Value
	varAddr, exists := g.symbolTable[varName]
	if !exists {
		g.ReportError("Tanımlanmamış değişken: %s", varName)
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, postfix ifadesi değerlendirilemiyor")
		return nil
	}

	// Mevcut değeri oku
	currentVal := g.currentBB.NewLoad(varAddr.Type().(*types.PointerType).ElemType, varAddr)

	// Operatöre göre işlem yap
	var newVal value.Value
	switch expr.Operator {
	case "++":
		// Artırma
		if types.IsInt(currentVal.Type()) {
			newVal = g.currentBB.NewAdd(currentVal, constant.NewInt(types.I32, 1))
		} else {
			g.ReportError("++ operatörü sadece integer tiplerinde kullanılabilir")
			return nil
		}
	case "--":
		// Azaltma
		if types.IsInt(currentVal.Type()) {
			newVal = g.currentBB.NewSub(currentVal, constant.NewInt(types.I32, 1))
		} else {
			g.ReportError("-- operatörü sadece integer tiplerinde kullanılabilir")
			return nil
		}
	default:
		g.ReportError("Desteklenmeyen postfix operatörü: %s", expr.Operator)
		return nil
	}

	// Yeni değeri değişkene ata
	g.currentBB.NewStore(newVal, varAddr)

	// Postfix operatörler eski değeri döndürür
	return currentVal
}

func (g *IRGenerator) generateInfixExpression(expr *ast.InfixExpression) value.Value {
	// ":=" operatörü için özel handling
	if expr.Operator == ":=" {
		// Sol taraf bir tanımlayıcı olmalı
		if ident, ok := expr.Left.(*ast.Identifier); ok {
			varName := ident.Value

			// Değişken zaten tanımlı mı kontrol et
			if _, exists := g.symbolTable[varName]; exists {
				g.ReportError("Değişken zaten tanımlı: %s", varName)
				return nil
			}

			// Sağ tarafı değerlendir
			right := g.generateExpression(expr.Right)
			if right == nil {
				return nil
			}

			// Sağ tarafın tipini belirle
			rightType := right.Type()

			if g.currentFunc == nil {
				g.ReportError("Kısa değişken tanımlama sadece fonksiyon içinde kullanılabilir")
				return nil
			}

			if g.currentBB == nil {
				g.ReportError("Geçerli bir blok yok, değişken tanımlanamıyor")
				return nil
			}

			// Değişken için bellek ayır
			alloca := g.currentBB.NewAlloca(rightType)
			alloca.SetName(varName)
			g.symbolTable[varName] = alloca

			// Değeri ata
			g.currentBB.NewStore(right, alloca)
			return right
		} else {
			g.ReportError("Kısa değişken tanımlama operatörünün sol tarafı bir tanımlayıcı olmalıdır")
			return nil
		}
	}

	// Diğer operatörler için normal işlem
	left := g.generateExpression(expr.Left)
	right := g.generateExpression(expr.Right)

	if left == nil || right == nil {
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, araek ifadesi değerlendirilemiyor")
		return nil
	}

	// Tip uyumluluğunu kontrol et ve gerekirse dönüşüm yap
	leftType := left.Type()
	rightType := right.Type()

	// Aritmetik ve atama operatörleri
	switch expr.Operator {
	case "+":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewAdd(left, right)
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFAdd(left, right)
		} else if g.isStringType(leftType) && g.isStringType(rightType) {
			// String concatenation
			return g.generateStringConcatenation(left, right)
		}
	case "-":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewSub(left, right)
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFSub(left, right)
		}
	case "*":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewMul(left, right)
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFMul(left, right)
		}
	case "/":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewSDiv(left, right) // Signed division
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFDiv(left, right)
		}
	case "=":
		// Atama operatörü
		// Sol taraf bir tanımlayıcı olmalı
		if ident, ok := expr.Left.(*ast.Identifier); ok {
			// Tanımlayıcının değerini sembol tablosundan bul
			if val, exists := g.symbolTable[ident.Value]; exists {
				// Değeri ata
				g.currentBB.NewStore(right, val)
				return right
			} else {
				g.ReportError("Tanımlanmamış tanımlayıcı: %s", ident.Value)
				return nil
			}
		} else {
			g.ReportError("Atama operatörünün sol tarafı bir tanımlayıcı olmalıdır")
			return nil
		}

	// Karşılaştırma operatörleri
	case "==":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewICmp(enum.IPredEQ, left, right)
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFCmp(enum.FPredOEQ, left, right)
		}
	case "!=":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewICmp(enum.IPredNE, left, right)
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFCmp(enum.FPredONE, left, right)
		}
	case "<":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewICmp(enum.IPredSLT, left, right) // Signed less than
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFCmp(enum.FPredOLT, left, right)
		}
	case ">":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewICmp(enum.IPredSGT, left, right) // Signed greater than
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFCmp(enum.FPredOGT, left, right)
		}
	case "<=":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewICmp(enum.IPredSLE, left, right) // Signed less equal
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFCmp(enum.FPredOLE, left, right)
		}
	case ">=":
		if types.IsInt(leftType) && types.IsInt(rightType) {
			return g.currentBB.NewICmp(enum.IPredSGE, left, right) // Signed greater equal
		} else if types.IsFloat(leftType) && types.IsFloat(rightType) {
			return g.currentBB.NewFCmp(enum.FPredOGE, left, right)
		}
	// Mantıksal operatörler
	case "&&":
		// Kısa devre değerlendirme için bloklar oluştur
		currFunc := g.currentFunc
		if currFunc == nil {
			g.ReportError("Geçerli bir fonksiyon yok, mantıksal AND değerlendirilemiyor")
			return nil
		}

		// Bloklar oluştur
		rightBlock := currFunc.NewBlock("")
		mergeBlock := currFunc.NewBlock("")

		// Sol ifade doğruysa sağ ifadeyi değerlendir, değilse direkt false döndür
		g.currentBB.NewCondBr(left, rightBlock, mergeBlock)

		// Sağ ifadeyi değerlendir
		g.currentBB = rightBlock
		g.generateExpression(expr.Right) // Sonucu kullanmıyoruz
		g.currentBB.NewBr(mergeBlock)

		// Sonuç bloğu
		g.currentBB = mergeBlock
		// Basitleştirilmiş yaklaşım: Sadece false döndür
		return constant.NewInt(types.I1, 0)
	case "||":
		// Kısa devre değerlendirme için bloklar oluştur
		currFunc := g.currentFunc
		if currFunc == nil {
			g.ReportError("Geçerli bir fonksiyon yok, mantıksal OR değerlendirilemiyor")
			return nil
		}

		// Bloklar oluştur
		rightBlock := currFunc.NewBlock("")
		mergeBlock := currFunc.NewBlock("")

		// Sol ifade yanlışsa sağ ifadeyi değerlendir, doğruysa direkt true döndür
		g.currentBB.NewCondBr(left, mergeBlock, rightBlock)

		// Sağ ifadeyi değerlendir
		g.currentBB = rightBlock
		g.generateExpression(expr.Right) // Sonucu kullanmıyoruz
		g.currentBB.NewBr(mergeBlock)

		// Sonuç bloğu
		g.currentBB = mergeBlock
		// Basitleştirilmiş yaklaşım: Sadece true döndür
		return constant.NewInt(types.I1, 1)
	}

	g.ReportError("Desteklenmeyen araek operatörü: %s", expr.Operator)
	return nil
}

func (g *IRGenerator) generateCallExpression(expr *ast.CallExpression) value.Value {
	var fn value.Value
	var funcName string

	// Fonksiyon türünü belirle
	switch f := expr.Function.(type) {
	case *ast.Identifier:
		// Normal function call: func()
		funcName = f.Value

		// Built-in functions için özel handling
		switch funcName {
		case "len":
			return g.generateLenCall(expr.Arguments)
		case "cap":
			return g.generateCapCall(expr.Arguments)
		case "append":
			return g.generateAppendCall(expr.Arguments)
		case "make":
			return g.generateMakeCall(expr.Arguments)
		}

		if val, exists := g.symbolTable[funcName]; exists {
			fn = val
		} else {
			// Fonksiyon bulunamadıysa, dış fonksiyon olarak tanımla
			fn = g.module.NewFunc(funcName, types.I32)
			g.symbolTable[funcName] = fn
		}
	case *ast.MemberExpression:
		// Member function call: package.func() veya object.method()
		return g.generateMemberFunctionCall(expr, f)
	default:
		g.ReportError("Desteklenmeyen fonksiyon çağrısı türü: %T", expr.Function)
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, fonksiyon çağrısı yapılamıyor")
		return nil
	}

	// Argümanları değerlendir
	args := make([]value.Value, 0, len(expr.Arguments))
	for _, arg := range expr.Arguments {
		argVal := g.generateExpression(arg)
		if argVal != nil {
			args = append(args, argVal)
		}
	}

	// Fonksiyon çağrısı yap
	return g.currentBB.NewCall(fn, args...)
}

// generateMemberFunctionCall, bir member function call için IR üretir.
func (g *IRGenerator) generateMemberFunctionCall(callExpr *ast.CallExpression, memberExpr *ast.MemberExpression) value.Value {
	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, member function çağrısı yapılamıyor")
		return nil
	}

	// Member adını al
	var memberName string
	if memberIdent, ok := memberExpr.Member.(*ast.Identifier); ok {
		memberName = memberIdent.Value
	} else {
		g.ReportError("Member adı bir tanımlayıcı olmalıdır")
		return nil
	}

	// Object adını al (package name için)
	var objectName string
	if objectIdent, ok := memberExpr.Object.(*ast.Identifier); ok {
		objectName = objectIdent.Value
	} else {
		g.ReportError("Object adı bir tanımlayıcı olmalıdır")
		return nil
	}

	// Package.function call olarak handle et (fmt.Println gibi)
	// Standard library functions için özel handling
	switch objectName {
	case "fmt":
		return g.generateFmtFunctionCall(memberName, callExpr.Arguments)
	case "os":
		return g.generateOsFunctionCall(memberName, callExpr.Arguments)
	default:
		// Diğer package'lar veya object method calls için
		// Şimdilik basit bir external function call olarak handle edelim
		fullFuncName := objectName + "_" + memberName

		// Fonksiyonu bul veya oluştur
		var fn value.Value
		if val, exists := g.symbolTable[fullFuncName]; exists {
			fn = val
		} else {
			// External function olarak tanımla
			fn = g.module.NewFunc(fullFuncName, types.I32)
			g.symbolTable[fullFuncName] = fn
		}

		// Argümanları değerlendir
		args := make([]value.Value, 0, len(callExpr.Arguments))
		for _, arg := range callExpr.Arguments {
			argVal := g.generateExpression(arg)
			if argVal != nil {
				args = append(args, argVal)
			}
		}

		// Fonksiyon çağrısı yap
		return g.currentBB.NewCall(fn, args...)
	}
}

// generateFmtFunctionCall, fmt package function call'ları için IR üretir.
func (g *IRGenerator) generateFmtFunctionCall(funcName string, args []ast.Expression) value.Value {
	switch funcName {
	case "Println", "Print", "Printf":
		// printf-style function olarak handle et
		return g.generatePrintfCall(funcName, args)
	default:
		g.ReportError("Desteklenmeyen fmt fonksiyonu: %s", funcName)
		return nil
	}
}

// generateOsFunctionCall, os package function call'ları için IR üretir.
func (g *IRGenerator) generateOsFunctionCall(funcName string, args []ast.Expression) value.Value {
	switch funcName {
	case "Exit":
		// exit function olarak handle et
		return g.generateExitCall(args)
	default:
		g.ReportError("Desteklenmeyen os fonksiyonu: %s", funcName)
		return nil
	}
}

// generatePrintfCall, printf-style function call'ları için IR üretir.
func (g *IRGenerator) generatePrintfCall(funcName string, args []ast.Expression) value.Value {
	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, printf çağrısı yapılamıyor")
		return nil
	}

	// Platform-specific print function kullan
	// Windows için puts, Linux/macOS için printf
	var printFunc *ir.Func
	var irArgs []value.Value

	if len(args) > 0 {
		// Multiple arguments için printf kullan
		if funcName == "Println" {
			// printf fonksiyonunu bul veya tanımla
			printFunc = g.getFunction("printf")
			if printFunc == nil {
				printFunc = g.module.NewFunc("printf", types.I32, ir.NewParam("format", types.NewPointer(types.I8)))
				printFunc.Sig.Variadic = true
				g.symbolTable["printf"] = printFunc
			}

			// Format string oluştur
			formatParts := make([]string, 0, len(args))
			for _, arg := range args {
				argVal := g.generateExpression(arg)
				if argVal != nil {
					// Argument tipine göre format belirle
					switch argVal.Type() {
					case types.I32, types.I64:
						formatParts = append(formatParts, "%d")
					case types.Float, types.Double:
						formatParts = append(formatParts, "%f")
					case types.I1:
						// Boolean değeri i32'ye extend et
						extendedVal := g.currentBB.NewZExt(argVal, types.I32)
						formatParts = append(formatParts, "%d")
						argVal = extendedVal
					default:
						formatParts = append(formatParts, "%s")
					}
					irArgs = append(irArgs, argVal)
				}
			}

			// Format string'i oluştur ve newline ekle
			formatString := strings.Join(formatParts, " ") + "\n"
			formatStr := g.generateStringLiteral(&ast.StringLiteral{Value: formatString})

			// Format string'i ilk argüman olarak ekle
			finalArgs := make([]value.Value, 0, len(irArgs)+1)
			finalArgs = append(finalArgs, formatStr)
			finalArgs = append(finalArgs, irArgs...)
			irArgs = finalArgs
		} else {
			// fmt.Print için - newline yok
			printFunc = g.getFunction("printf")
			if printFunc == nil {
				printFunc = g.module.NewFunc("printf", types.I32, ir.NewParam("format", types.NewPointer(types.I8)))
				printFunc.Sig.Variadic = true
				g.symbolTable["printf"] = printFunc
			}

			// Format string oluştur (newline olmadan)
			formatParts := make([]string, 0, len(args))
			for _, arg := range args {
				argVal := g.generateExpression(arg)
				if argVal != nil {
					// Argument tipine göre format belirle
					switch argVal.Type() {
					case types.I32, types.I64:
						formatParts = append(formatParts, "%d")
					case types.Float, types.Double:
						formatParts = append(formatParts, "%f")
					case types.I1:
						// Boolean değeri i32'ye extend et
						extendedVal := g.currentBB.NewZExt(argVal, types.I32)
						formatParts = append(formatParts, "%d")
						argVal = extendedVal
					default:
						formatParts = append(formatParts, "%s")
					}
					irArgs = append(irArgs, argVal)
				}
			}

			// Format string'i oluştur
			formatString := strings.Join(formatParts, " ")
			formatStr := g.generateStringLiteral(&ast.StringLiteral{Value: formatString})

			// Format string'i ilk argüman olarak ekle
			finalArgs := make([]value.Value, 0, len(irArgs)+1)
			finalArgs = append(finalArgs, formatStr)
			finalArgs = append(finalArgs, irArgs...)
			irArgs = finalArgs
		}
	} else {
		// Argüman yoksa sadece newline yazdır (Println için)
		if funcName == "Println" {
			printFunc = g.getFunction("puts")
			if printFunc == nil {
				printFunc = g.module.NewFunc("puts", types.I32, ir.NewParam("str", types.NewPointer(types.I8)))
				g.symbolTable["puts"] = printFunc
			}
			// Boş string için newline
			emptyStr := g.generateStringLiteral(&ast.StringLiteral{Value: ""})
			irArgs = append(irArgs, emptyStr)
		}
	}

	if printFunc == nil {
		g.ReportError("Print fonksiyonu oluşturulamadı")
		return nil
	}

	// Print fonksiyonu çağrısı yap
	return g.currentBB.NewCall(printFunc, irArgs...)
}

// generateExitCall, exit function call'ı için IR üretir.
func (g *IRGenerator) generateExitCall(args []ast.Expression) value.Value {
	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, exit çağrısı yapılamıyor")
		return nil
	}

	// exit fonksiyonunu tanımla (eğer yoksa)
	exitFunc := g.module.NewFunc("exit", types.Void, ir.NewParam("status", types.I32))

	// Argümanı değerlendir
	var exitCode value.Value
	if len(args) > 0 {
		exitCode = g.generateExpression(args[0])
	} else {
		exitCode = constant.NewInt(types.I32, 0) // Varsayılan exit code
	}

	// exit çağrısı yap
	return g.currentBB.NewCall(exitFunc, exitCode)
}

func (g *IRGenerator) generateFunctionLiteral(expr *ast.FunctionLiteral) value.Value {
	// Fonksiyon adını belirle
	funcName := "anonymous_func"

	// Parametre tiplerini belirle
	paramTypes := make([]types.Type, len(expr.Parameters))
	for i := range expr.Parameters {
		paramTypes[i] = types.I32 // Varsayılan olarak int32
	}

	// Dönüş tipini belirle
	var returnType types.Type = types.I32 // Varsayılan olarak int32

	// Fonksiyonu oluştur
	fn := g.module.NewFunc(funcName, returnType)

	// Parametreleri ekle
	for _, paramType := range paramTypes {
		param := ir.NewParam("", paramType)
		fn.Params = append(fn.Params, param)
	}

	// Önceki durumu kaydet
	prevFunc := g.currentFunc
	prevBB := g.currentBB

	// Yeni durumu ayarla
	g.currentFunc = fn
	entryBlock := fn.NewBlock("entry")
	g.currentBB = entryBlock

	// Parametreleri sembol tablosuna ekle
	if len(expr.Parameters) > 0 && len(fn.Params) > 0 {
		for i, param := range expr.Parameters {
			if i < len(fn.Params) {
				paramName := param.Value
				paramVal := fn.Params[i]
				paramVal.SetName(paramName)

				// Parametre için yerel değişken oluştur
				alloca := entryBlock.NewAlloca(paramTypes[i])
				alloca.SetName(paramName + ".addr")
				entryBlock.NewStore(paramVal, alloca)

				g.symbolTable[paramName] = alloca
			}
		}
	}

	// Fonksiyon gövdesini işle
	if expr.Body != nil {
		g.generateBlockStatement(expr.Body)
	}

	// Eğer son blok bir dönüş ifadesi ile bitmiyorsa, varsayılan dönüş ekle
	if g.currentBB.Term == nil {
		g.currentBB.NewRet(constant.NewInt(types.I32, 0))
	}

	// Önceki durumu geri yükle
	g.currentFunc = prevFunc
	g.currentBB = prevBB

	return fn
}

// Deyim türleri için IR üretme fonksiyonları

func (g *IRGenerator) generateVarStatement(stmt *ast.VarStatement) {
	// Değişken adını al
	varName := stmt.Name.Value

	// Değişken tipini belirle
	var varType types.Type
	if stmt.Type != nil {
		// Tip belirtilmişse, bu tipi kullan
		if typeIdent, ok := stmt.Type.(*ast.Identifier); ok {
			if t, exists := g.typeTable[typeIdent.Value]; exists {
				varType = t
			} else {
				g.ReportError("Bilinmeyen tip: %s", typeIdent.Value)
				return
			}
		} else {
			g.ReportError("Desteklenmeyen tip ifadesi: %T", stmt.Type)
			return
		}
	} else if stmt.Value != nil {
		// Tip belirtilmemişse ve değer varsa, değerin tipini kullan
		exprType := g.getExpressionType(stmt.Value)
		if exprType != nil {
			varType = exprType
		} else {
			g.ReportError("Değişken tipi belirlenemedi: %s", varName)
			return
		}
	} else {
		g.ReportError("Değişken tipi belirtilmemiş ve değer atanmamış: %s", varName)
		return
	}

	// Değişken global mi yoksa lokal mi?
	if g.currentFunc == nil {
		// Global değişken
		globalVar := g.module.NewGlobalDef(varName, constant.NewZeroInitializer(varType))
		g.symbolTable[varName] = globalVar

		// Değer atanmışsa, değeri ata
		if stmt.Value != nil {
			if constVal := g.generateConstantExpression(stmt.Value); constVal != nil {
				globalVar.Init = constVal
			}
		}

		// Hata ayıklama bilgisi ekle
		if g.generateDebug {
			// Global değişken için hata ayıklama bilgisi oluştur
			// Not: LLVM IR'da global değişkenler için hata ayıklama bilgisi ekleme
			// işlemi daha karmaşıktır ve bu örnekte basitleştirilmiştir.
		}
	} else {
		// Lokal değişken
		if g.currentBB == nil {
			g.ReportError("Geçerli bir blok yok, değişken tanımlanamıyor: %s", varName)
			return
		}

		// Değişken için bellek ayır
		alloca := g.currentBB.NewAlloca(varType)
		alloca.SetName(varName)
		g.symbolTable[varName] = alloca

		// Hata ayıklama bilgisi ekle
		if g.generateDebug {
			// TODO: Debug API değişikliği nedeniyle geçici olarak devre dışı
			// pos := stmt.Pos()
			// localVar := g.debugInfo.CreateLocalVariable(...)
			// g.debugInfo.InsertDeclare(...)
		}

		// Değer atanmışsa, değeri ata
		if stmt.Value != nil {
			// Hata ayıklama bilgisi için konum bilgisini ayarla
			if g.generateDebug && stmt.Value.Pos().IsValid() {
				pos := stmt.Value.Pos()
				g.debugInfo.SetLocation(pos.Line, pos.Column, g.sourceFile)
			}

			val := g.generateExpression(stmt.Value)
			if val != nil {
				g.currentBB.NewStore(val, alloca)
			}
		}
	}
}

func (g *IRGenerator) generateReturnStatement(stmt *ast.ReturnStatement) {
	if g.currentFunc == nil {
		g.ReportError("Fonksiyon dışında dönüş deyimi kullanılamaz")
		return
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, dönüş deyimi değerlendirilemiyor")
		return
	}

	// Hata ayıklama bilgisi için konum bilgisini ayarla
	if g.generateDebug && stmt.Pos().IsValid() {
		pos := stmt.Pos()
		g.debugInfo.SetLocation(pos.Line, pos.Column, g.sourceFile)
	}

	// Dönüş değeri varsa değerlendir
	if stmt.ReturnValue != nil {
		retVal := g.generateExpression(stmt.ReturnValue)
		if retVal != nil {
			g.currentBB.NewRet(retVal)
		} else {
			g.currentBB.NewRet(constant.NewInt(types.I32, 0)) // Varsayılan dönüş değeri
		}
	} else {
		// Dönüş değeri yoksa void dönüş
		g.currentBB.NewRet(constant.NewInt(types.I32, 0)) // Varsayılan dönüş değeri
	}
}

func (g *IRGenerator) generateBlockStatement(stmt *ast.BlockStatement) {
	// Hata ayıklama bilgisi için sözcüksel blok oluştur
	if g.generateDebug && stmt.Pos().IsValid() {
		// TODO: Debug API değişikliği nedeniyle geçici olarak devre dışı
		// pos := stmt.Pos()
		// g.debugInfo.CreateLexicalBlock(...)
	}

	// Blok içindeki tüm deyimleri değerlendir
	for _, s := range stmt.Statements {
		// Hata ayıklama bilgisi için konum bilgisini ayarla
		if g.generateDebug && s.Pos().IsValid() {
			pos := s.Pos()
			g.debugInfo.SetLocation(pos.Line, pos.Column, g.sourceFile)
		}

		g.generateStatement(s)

		// Eğer bir dönüş deyimi ile karşılaşıldıysa, sonraki deyimleri değerlendirme
		if g.currentBB != nil && g.currentBB.Term != nil {
			break
		}
	}

	// Sözcüksel bloğu kapat
	if g.generateDebug {
		// TODO: Debug API değişikliği nedeniyle geçici olarak devre dışı
		// g.debugInfo.FinishLexicalBlock()
	}
}

// Fonksiyon tanımlamaları için özel bir fonksiyon eklenebilir
// Şimdilik bu fonksiyonu kaldırıyoruz

// generateIfExpression, bir if ifadesi için IR üretir.
func (g *IRGenerator) generateIfExpression(expr *ast.IfExpression) value.Value {
	// Koşul ifadesini değerlendir
	condition := g.generateExpression(expr.Condition)
	if condition == nil {
		return nil
	}

	if g.currentFunc == nil {
		g.ReportError("Geçerli bir fonksiyon yok, if ifadesi değerlendirilemiyor")
		return nil
	}

	// Unique label'lar oluştur
	g.labelCounter++
	labelSuffix := fmt.Sprintf("%d", g.labelCounter)

	// Bloklar oluştur
	thenBlock := g.currentFunc.NewBlock("if.then." + labelSuffix)
	elseBlock := g.currentFunc.NewBlock("if.else." + labelSuffix)
	mergeBlock := g.currentFunc.NewBlock("if.end." + labelSuffix)

	// Koşula göre dallanma
	g.currentBB.NewCondBr(condition, thenBlock, elseBlock)

	// Then bloğunu işle
	g.currentBB = thenBlock
	if expr.Consequence != nil {
		g.generateBlockStatement(expr.Consequence)
		// Eğer blok bir dönüş ifadesi ile bitmiyorsa, merge bloğuna git
		if g.currentBB.Term == nil {
			g.currentBB.NewBr(mergeBlock)
		}
	} else {
		g.currentBB.NewBr(mergeBlock)
	}

	// Else bloğunu işle
	g.currentBB = elseBlock
	if expr.Alternative != nil {
		g.generateBlockStatement(expr.Alternative)
		// Eğer blok bir dönüş ifadesi ile bitmiyorsa, merge bloğuna git
		if g.currentBB.Term == nil {
			g.currentBB.NewBr(mergeBlock)
		}
	} else {
		g.currentBB.NewBr(mergeBlock)
	}

	// Merge bloğuna geç
	g.currentBB = mergeBlock

	// If ifadesi bir değer döndürmez, sadece kontrol akışını değiştirir
	return nil
}

// generateWhileStatement, bir while döngüsü için IR üretir.
func (g *IRGenerator) generateWhileStatement(stmt *ast.WhileStatement) {
	if g.currentFunc == nil {
		g.ReportError("Geçerli bir fonksiyon yok, while döngüsü değerlendirilemiyor")
		return
	}

	// Bloklar oluştur
	condBlock := g.currentFunc.NewBlock("while.cond")
	bodyBlock := g.currentFunc.NewBlock("while.body")
	endBlock := g.currentFunc.NewBlock("while.end")

	// Koşul bloğuna git
	g.currentBB.NewBr(condBlock)

	// Koşul bloğunu işle
	g.currentBB = condBlock
	condition := g.generateExpression(stmt.Condition)
	if condition == nil {
		return
	}

	// Koşula göre dallanma
	g.currentBB.NewCondBr(condition, bodyBlock, endBlock)

	// Döngü gövdesini işle
	g.currentBB = bodyBlock
	if stmt.Body != nil {
		g.generateBlockStatement(stmt.Body)
	}

	// Koşul bloğuna geri dön
	if g.currentBB.Term == nil {
		g.currentBB.NewBr(condBlock)
	}

	// Döngü sonrası bloğa geç
	g.currentBB = endBlock
}

// generateForStatement, bir for döngüsü için IR üretir.
func (g *IRGenerator) generateForStatement(stmt *ast.ForStatement) {
	if g.currentFunc == nil {
		g.ReportError("Geçerli bir fonksiyon yok, for döngüsü değerlendirilemiyor")
		return
	}

	// Unique label'lar oluştur
	g.labelCounter++
	labelSuffix := fmt.Sprintf("%d", g.labelCounter)

	// Bloklar oluştur
	initBlock := g.currentFunc.NewBlock("for.init." + labelSuffix)
	condBlock := g.currentFunc.NewBlock("for.cond." + labelSuffix)
	bodyBlock := g.currentFunc.NewBlock("for.body." + labelSuffix)
	postBlock := g.currentFunc.NewBlock("for.post." + labelSuffix)
	endBlock := g.currentFunc.NewBlock("for.end." + labelSuffix)

	// Init bloğuna git
	g.currentBB.NewBr(initBlock)

	// Init bloğunu işle
	g.currentBB = initBlock
	if stmt.Init != nil {
		g.generateStatement(stmt.Init)
	}
	g.currentBB.NewBr(condBlock)

	// Koşul bloğunu işle
	g.currentBB = condBlock
	if stmt.Condition != nil {
		condition := g.generateExpression(stmt.Condition)
		if condition == nil {
			return
		}
		// Koşula göre dallanma
		g.currentBB.NewCondBr(condition, bodyBlock, endBlock)
	} else {
		// Koşul yoksa sonsuz döngü (body'ye git)
		g.currentBB.NewBr(bodyBlock)
	}

	// Döngü gövdesini işle
	g.currentBB = bodyBlock
	if stmt.Body != nil {
		g.generateBlockStatement(stmt.Body)
	}
	// Body'den post bloğuna git
	if g.currentBB.Term == nil {
		g.currentBB.NewBr(postBlock)
	}

	// Post bloğunu işle
	g.currentBB = postBlock
	if stmt.Post != nil {
		g.generateStatement(stmt.Post)
	}
	// Post'tan koşul bloğuna geri dön
	if g.currentBB.Term == nil {
		g.currentBB.NewBr(condBlock)
	}

	// Döngü sonrası bloğa geç
	g.currentBB = endBlock
}

// generateSwitchStatement, bir switch ifadesi için IR üretir.
func (g *IRGenerator) generateSwitchStatement(stmt *ast.SwitchStatement) {
	if g.currentFunc == nil {
		g.ReportError("Geçerli bir fonksiyon yok, switch ifadesi değerlendirilemiyor")
		return
	}

	// Unique label'lar oluştur
	g.labelCounter++
	labelSuffix := fmt.Sprintf("%d", g.labelCounter)

	// Switch tag'ini değerlendir (varsa)
	var switchValue value.Value
	if stmt.Tag != nil {
		switchValue = g.generateExpression(stmt.Tag)
		if switchValue == nil {
			return
		}
	}

	// Bloklar oluştur
	endBlock := g.currentFunc.NewBlock("switch.end." + labelSuffix)
	var defaultBlock *ir.Block
	caseBlocks := make([]*ir.Block, len(stmt.Cases))

	// Her case için blok oluştur
	for i, caseClause := range stmt.Cases {
		if caseClause.Token.Type == token.DEFAULT {
			defaultBlock = g.currentFunc.NewBlock("switch.default." + labelSuffix)
			caseBlocks[i] = defaultBlock
		} else {
			caseBlocks[i] = g.currentFunc.NewBlock(fmt.Sprintf("switch.case.%d.%s", i, labelSuffix))
		}
	}

	// Eğer default blok yoksa, end bloğuna git
	if defaultBlock == nil {
		defaultBlock = endBlock
	}

	// Switch logic'i implement et
	if switchValue != nil {
		// Tag'li switch: her case değerini kontrol et
		g.generateTaggedSwitch(stmt, switchValue, caseBlocks, defaultBlock, endBlock)
	} else {
		// Tag'siz switch: boolean case'ler
		g.generateBooleanSwitch(stmt, caseBlocks, defaultBlock, endBlock)
	}

	// End bloğuna geç
	g.currentBB = endBlock
}

// generateTaggedSwitch, tag'li switch için IR üretir.
func (g *IRGenerator) generateTaggedSwitch(stmt *ast.SwitchStatement, switchValue value.Value,
	caseBlocks []*ir.Block, defaultBlock *ir.Block, endBlock *ir.Block) {

	currentBlock := g.currentBB

	// Her case için karşılaştırma yap
	for i, caseClause := range stmt.Cases {
		if caseClause.Token.Type == token.DEFAULT {
			continue // Default case'i sonra işleyeceğiz
		}

		// Case değerlerini kontrol et
		for _, caseValue := range caseClause.Values {
			val := g.generateExpression(caseValue)
			if val == nil {
				continue
			}

			// Karşılaştırma yap
			cmp := currentBlock.NewICmp(enum.IPredEQ, switchValue, val)

			// Sonraki karşılaştırma için yeni blok oluştur
			nextBlock := g.currentFunc.NewBlock(fmt.Sprintf("switch.next.%d", i))

			// Eşitse case bloğuna, değilse sonraki karşılaştırmaya git
			currentBlock.NewCondBr(cmp, caseBlocks[i], nextBlock)
			currentBlock = nextBlock
		}
	}

	// Hiçbir case eşleşmezse default'a git
	currentBlock.NewBr(defaultBlock)

	// Case bloklarını işle
	for i, caseClause := range stmt.Cases {
		g.currentBB = caseBlocks[i]

		// Case body'sini işle
		for _, bodyStmt := range caseClause.Body {
			g.generateStatement(bodyStmt)

			// Eğer return statement varsa, sonraki statement'ları işleme
			if g.currentBB.Term != nil {
				break
			}
		}

		// Fallthrough kontrolü - Go'da varsayılan olarak break var
		if g.currentBB.Term == nil {
			// Explicit fallthrough statement yoksa end bloğuna git
			g.currentBB.NewBr(endBlock)
		}
	}
}

// generateBooleanSwitch, tag'siz switch için IR üretir.
func (g *IRGenerator) generateBooleanSwitch(stmt *ast.SwitchStatement,
	caseBlocks []*ir.Block, defaultBlock *ir.Block, endBlock *ir.Block) {

	currentBlock := g.currentBB

	// Her case'i if-else chain olarak implement et
	for i, caseClause := range stmt.Cases {
		if caseClause.Token.Type == token.DEFAULT {
			continue // Default case'i sonra işleyeceğiz
		}

		// Case değerlerini OR ile birleştir
		var condition value.Value
		for j, caseValue := range caseClause.Values {
			val := g.generateExpression(caseValue)
			if val == nil {
				continue
			}

			if j == 0 {
				condition = val
			} else {
				// OR ile birleştir
				condition = currentBlock.NewOr(condition, val)
			}
		}

		if condition != nil {
			// Sonraki case için blok oluştur
			nextBlock := g.currentFunc.NewBlock(fmt.Sprintf("switch.next.%d", i))

			// Koşul doğruysa case bloğuna, değilse sonraki case'e git
			currentBlock.NewCondBr(condition, caseBlocks[i], nextBlock)
			currentBlock = nextBlock
		}
	}

	// Hiçbir case eşleşmezse default'a git
	currentBlock.NewBr(defaultBlock)

	// Case bloklarını işle (tagged switch ile aynı)
	for i, caseClause := range stmt.Cases {
		g.currentBB = caseBlocks[i]

		// Case body'sini işle
		for _, bodyStmt := range caseClause.Body {
			g.generateStatement(bodyStmt)

			// Eğer return statement varsa, sonraki statement'ları işleme
			if g.currentBB.Term != nil {
				break
			}
		}

		// Fallthrough kontrolü
		if g.currentBB.Term == nil {
			g.currentBB.NewBr(endBlock)
		}
	}
}

// generateArrayLiteral, bir array literal için IR üretir.
func (g *IRGenerator) generateArrayLiteral(expr *ast.ArrayLiteral) value.Value {
	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, array literal değerlendirilemiyor")
		return nil
	}

	// Array element sayısını belirle
	elementCount := len(expr.Elements)
	if elementCount == 0 {
		// Boş array için null pointer döndür
		return constant.NewNull(types.NewPointer(types.I32))
	}

	// İlk element'in tipini belirle (tüm elementler aynı tipte olmalı)
	firstElement := g.generateExpression(expr.Elements[0])
	if firstElement == nil {
		return nil
	}
	elementType := firstElement.Type()

	// Array tipini oluştur
	arrayType := types.NewArray(uint64(elementCount), elementType)

	// Stack'te array için yer ayır
	arrayAlloca := g.currentBB.NewAlloca(arrayType)

	// Array elementlerini doldur
	for i, element := range expr.Elements {
		elementValue := g.generateExpression(element)
		if elementValue == nil {
			continue
		}

		// Element'in tipini kontrol et
		if !elementValue.Type().Equal(elementType) {
			g.ReportError("Array element tipi uyumsuz: beklenen %s, alınan %s",
				elementType.String(), elementValue.Type().String())
			continue
		}

		// Array index'ini hesapla
		indices := []value.Value{
			constant.NewInt(types.I32, 0),        // Array pointer
			constant.NewInt(types.I32, int64(i)), // Element index
		}

		// Element adresini al
		elementPtr := g.currentBB.NewGetElementPtr(arrayType, arrayAlloca, indices...)

		// Element'i store et
		g.currentBB.NewStore(elementValue, elementPtr)
	}

	// Array pointer'ını döndür
	return arrayAlloca
}

// generateIndexExpression, bir array/slice indexing için IR üretir.
func (g *IRGenerator) generateIndexExpression(expr *ast.IndexExpression) value.Value {
	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, index expression değerlendirilemiyor")
		return nil
	}

	// Array/slice değerini al
	arrayValue := g.generateExpression(expr.Left)
	if arrayValue == nil {
		return nil
	}

	// Index değerini al
	indexValue := g.generateExpression(expr.Index)
	if indexValue == nil {
		return nil
	}

	// Index'in integer olduğunu kontrol et
	if !types.IsInt(indexValue.Type()) {
		g.ReportError("Array index integer olmalıdır, alınan: %s", indexValue.Type().String())
		return nil
	}

	// Array tipini kontrol et
	arrayType, ok := arrayValue.Type().(*types.PointerType)
	if !ok {
		g.ReportError("Index expression sadece array/slice'larda kullanılabilir")
		return nil
	}

	// Element tipini belirle
	var elementType types.Type
	var arrayLength value.Value

	if arrType, ok := arrayType.ElemType.(*types.ArrayType); ok {
		// Static array
		elementType = arrType.ElemType
		arrayLength = constant.NewInt(types.I32, int64(arrType.Len))
	} else if structType, ok := arrayType.ElemType.(*types.StructType); ok {
		// Slice struct: {data *T, len int32, cap int32}
		if len(structType.Fields) >= 2 {
			if dataPtr, ok := structType.Fields[0].(*types.PointerType); ok {
				elementType = dataPtr.ElemType
			}
			// len field'ını al (index 1)
			lenIndices := []value.Value{
				constant.NewInt(types.I32, 0),
				constant.NewInt(types.I32, 1),
			}
			lenPtr := g.currentBB.NewGetElementPtr(structType, arrayValue, lenIndices...)
			arrayLength = g.currentBB.NewLoad(types.I32, lenPtr)
		}
	} else {
		elementType = arrayType.ElemType
		// Bilinmeyen uzunluk için bounds checking yapma
		arrayLength = nil
	}

	// Runtime bounds checking
	if arrayLength != nil {
		g.generateBoundsCheck(indexValue, arrayLength)
	}

	// Element adresini hesapla
	var elementPtr value.Value
	if structType, ok := arrayType.ElemType.(*types.StructType); ok {
		// Slice için: data pointer'ını al ve index'le
		dataIndices := []value.Value{
			constant.NewInt(types.I32, 0),
			constant.NewInt(types.I32, 0), // data field
		}
		dataPtr := g.currentBB.NewGetElementPtr(structType, arrayValue, dataIndices...)
		dataArray := g.currentBB.NewLoad(types.NewPointer(elementType), dataPtr)
		elementPtr = g.currentBB.NewGetElementPtr(elementType, dataArray, indexValue)
	} else {
		// Array için: direkt indexing
		indices := []value.Value{
			constant.NewInt(types.I32, 0), // Array pointer
			indexValue,                    // Element index
		}
		elementPtr = g.currentBB.NewGetElementPtr(arrayType.ElemType, arrayValue, indices...)
	}

	// Element değerini load et
	return g.currentBB.NewLoad(elementType, elementPtr)
}

// generateFunctionStatement, bir fonksiyon tanımlaması için IR üretir.
func (g *IRGenerator) generateFunctionStatement(stmt *ast.FunctionStatement) {
	// Fonksiyon adını al
	funcName := stmt.Name.Value

	// Parametre tiplerini belirle
	paramTypes := make([]types.Type, len(stmt.Parameters))
	for i := range stmt.Parameters {
		// TODO: Parametre tip sistemi implement edilecek
		paramTypes[i] = types.I32 // Varsayılan olarak int32
	}

	// Dönüş tipini belirle
	var returnType types.Type = types.I32 // Varsayılan olarak int32
	if stmt.ReturnType != nil {
		if typeIdent, ok := stmt.ReturnType.(*ast.Identifier); ok {
			if t, exists := g.typeTable[typeIdent.Value]; exists {
				returnType = t
			} else {
				g.ReportError("Bilinmeyen tip: %s", typeIdent.Value)
			}
		} else {
			g.ReportError("Desteklenmeyen tip ifadesi: %T", stmt.ReturnType)
		}
	}

	// Fonksiyonu oluştur
	fn := g.module.NewFunc(funcName, returnType)

	// Parametreleri ekle
	for i, param := range stmt.Parameters {
		paramName := param.Value
		fn.Params = append(fn.Params, ir.NewParam(paramName, paramTypes[i]))
	}

	// Hata ayıklama bilgisi ekle
	if g.generateDebug {
		// Fonksiyon için hata ayıklama bilgisi oluştur
		file := g.debugInfo.getOrCreateFileMetadata(g.sourceFile, g.sourceDir)
		pos := stmt.Pos()
		g.debugInfo.CreateFunction(
			fn,
			funcName,
			funcName,
			file,
			pos.Line,
			false,
			true,
			pos.Line,
			0, // Flags
			false,
		)
	}

	// Önceki durumu kaydet
	prevFunc := g.currentFunc
	prevBB := g.currentBB

	// Yeni durumu ayarla
	g.currentFunc = fn
	entryBlock := fn.NewBlock("entry")
	g.currentBB = entryBlock

	// Parametreleri sembol tablosuna ekle
	for i, param := range stmt.Parameters {
		paramName := param.Value
		paramVal := fn.Params[i]

		// Parametre için yerel değişken oluştur
		alloca := entryBlock.NewAlloca(paramTypes[i])
		alloca.SetName(paramName + ".addr")
		entryBlock.NewStore(paramVal, alloca)

		// Hata ayıklama bilgisi ekle
		if g.generateDebug {
			// TODO: Debug API değişikliği nedeniyle geçici olarak devre dışı
			// pos := param.Pos()
			// localVar := g.debugInfo.CreateLocalVariable(...)
			// g.debugInfo.InsertDeclare(...)
		}

		g.symbolTable[paramName] = alloca
	}

	// Fonksiyon gövdesini işle
	if stmt.Body != nil {
		// Hata ayıklama bilgisi için sözcüksel blok oluştur
		if g.generateDebug {
			// TODO: Debug API değişikliği nedeniyle geçici olarak devre dışı
			// pos := stmt.Body.Pos()
			// g.debugInfo.CreateLexicalBlock(...)
		}

		g.generateBlockStatement(stmt.Body)

		// Sözcüksel bloğu kapat
		if g.generateDebug {
			// TODO: Debug API değişikliği nedeniyle geçici olarak devre dışı
			// g.debugInfo.FinishLexicalBlock()
		}
	}

	// Eğer son blok bir dönüş ifadesi ile bitmiyorsa, varsayılan dönüş ekle
	if g.currentBB.Term == nil {
		// Hata ayıklama bilgisi için konum bilgisini ayarla
		if g.generateDebug {
			g.debugInfo.SetLocation(stmt.Body.End().Line, stmt.Body.End().Column, g.sourceFile)
		}

		g.currentBB.NewRet(constant.NewInt(types.I32, 0))
	}

	// Fonksiyon hata ayıklama bilgisini tamamla
	if g.generateDebug {
		// TODO: Debug API değişikliği nedeniyle geçici olarak devre dışı
		// g.debugInfo.FinishFunction()
	}

	// Önceki durumu geri yükle
	g.currentFunc = prevFunc
	g.currentBB = prevBB

	// Fonksiyonu sembol tablosuna ekle
	g.symbolTable[funcName] = fn
}

// Built-in functions implementation

// generateLenCall, len() built-in function için IR üretir.
func (g *IRGenerator) generateLenCall(args []ast.Expression) value.Value {
	if len(args) != 1 {
		g.ReportError("len() fonksiyonu tam olarak 1 argüman alır, %d verildi", len(args))
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, len() çağrısı yapılamıyor")
		return nil
	}

	// Argümanı değerlendir
	arg := g.generateExpression(args[0])
	if arg == nil {
		return nil
	}

	// Argüman tipini kontrol et
	argType := arg.Type()
	if ptrType, ok := argType.(*types.PointerType); ok {
		if arrType, ok := ptrType.ElemType.(*types.ArrayType); ok {
			// Array için: sabit uzunluk döndür
			return constant.NewInt(types.I32, int64(arrType.Len))
		} else {
			// Slice için: runtime length hesapla
			// Slice struct: {data *T, len int32, cap int32}
			// len field'ını al (index 1)
			indices := []value.Value{
				constant.NewInt(types.I32, 0), // Struct pointer
				constant.NewInt(types.I32, 1), // len field
			}
			lenPtr := g.currentBB.NewGetElementPtr(ptrType.ElemType, arg, indices...)
			return g.currentBB.NewLoad(types.I32, lenPtr)
		}
	}

	g.ReportError("len() fonksiyonu sadece array ve slice'larda kullanılabilir")
	return nil
}

// generateCapCall, cap() built-in function için IR üretir.
func (g *IRGenerator) generateCapCall(args []ast.Expression) value.Value {
	if len(args) != 1 {
		g.ReportError("cap() fonksiyonu tam olarak 1 argüman alır, %d verildi", len(args))
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, cap() çağrısı yapılamıyor")
		return nil
	}

	// Argümanı değerlendir
	arg := g.generateExpression(args[0])
	if arg == nil {
		return nil
	}

	// Argüman tipini kontrol et
	argType := arg.Type()
	if ptrType, ok := argType.(*types.PointerType); ok {
		if arrType, ok := ptrType.ElemType.(*types.ArrayType); ok {
			// Array için: sabit uzunluk döndür (len == cap)
			return constant.NewInt(types.I32, int64(arrType.Len))
		} else {
			// Slice için: runtime capacity hesapla
			// Slice struct: {data *T, len int32, cap int32}
			// cap field'ını al (index 2)
			indices := []value.Value{
				constant.NewInt(types.I32, 0), // Struct pointer
				constant.NewInt(types.I32, 2), // cap field
			}
			capPtr := g.currentBB.NewGetElementPtr(ptrType.ElemType, arg, indices...)
			return g.currentBB.NewLoad(types.I32, capPtr)
		}
	}

	g.ReportError("cap() fonksiyonu sadece array ve slice'larda kullanılabilir")
	return nil
}

// generateAppendCall, append() built-in function için IR üretir.
func (g *IRGenerator) generateAppendCall(args []ast.Expression) value.Value {
	if len(args) < 2 {
		g.ReportError("append() fonksiyonu en az 2 argüman alır, %d verildi", len(args))
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, append() çağrısı yapılamıyor")
		return nil
	}

	// İlk argüman slice olmalı
	sliceArg := g.generateExpression(args[0])
	if sliceArg == nil {
		return nil
	}

	// Slice tipini kontrol et
	sliceType := sliceArg.Type()
	ptrType, ok := sliceType.(*types.PointerType)
	if !ok {
		g.ReportError("append() fonksiyonunun ilk argümanı slice olmalıdır")
		return nil
	}

	// Element tipini belirle
	var elementType types.Type
	if structType, ok := ptrType.ElemType.(*types.StructType); ok {
		// Slice struct: {data *T, len int32, cap int32}
		if len(structType.Fields) >= 1 {
			if dataPtr, ok := structType.Fields[0].(*types.PointerType); ok {
				elementType = dataPtr.ElemType
			}
		}
	}

	if elementType == nil {
		g.ReportError("Slice element tipi belirlenemedi")
		return nil
	}

	// Mevcut slice bilgilerini al
	// len field (index 1)
	lenIndices := []value.Value{
		constant.NewInt(types.I32, 0),
		constant.NewInt(types.I32, 1),
	}
	lenPtr := g.currentBB.NewGetElementPtr(ptrType.ElemType, sliceArg, lenIndices...)
	currentLen := g.currentBB.NewLoad(types.I32, lenPtr)

	// cap field (index 2)
	capIndices := []value.Value{
		constant.NewInt(types.I32, 0),
		constant.NewInt(types.I32, 2),
	}
	capPtr := g.currentBB.NewGetElementPtr(ptrType.ElemType, sliceArg, capIndices...)
	currentCap := g.currentBB.NewLoad(types.I32, capPtr)

	// Eklenecek element sayısını hesapla
	appendCount := constant.NewInt(types.I32, int64(len(args)-1))
	newLen := g.currentBB.NewAdd(currentLen, appendCount)

	// Capacity kontrolü - basit implementasyon
	needsRealloc := g.currentBB.NewICmp(enum.IPredSGT, newLen, currentCap)

	// Conditional blocks oluştur
	reallocBlock := g.currentFunc.NewBlock("realloc")
	appendBlock := g.currentFunc.NewBlock("append")
	endBlock := g.currentFunc.NewBlock("append_end")

	g.currentBB.NewCondBr(needsRealloc, reallocBlock, appendBlock)

	// Reallocation block
	g.currentBB = reallocBlock
	// Basit capacity doubling strategy
	doubleCap := g.currentBB.NewMul(currentCap, constant.NewInt(types.I32, 2))
	// Minimum capacity check
	minCap := g.currentBB.NewICmp(enum.IPredSLT, doubleCap, newLen)
	newCap := g.currentBB.NewSelect(minCap, newLen, doubleCap)

	// TODO: Gerçek memory reallocation implementasyonu
	// Şimdilik sadece capacity'yi güncelle
	g.currentBB.NewStore(newCap, capPtr)
	g.currentBB.NewBr(appendBlock)

	// Append block
	g.currentBB = appendBlock
	// Yeni elementleri ekle
	for i, argExpr := range args[1:] {
		elementVal := g.generateExpression(argExpr)
		if elementVal == nil {
			continue
		}

		// Element index'ini hesapla
		elementIndex := g.currentBB.NewAdd(currentLen, constant.NewInt(types.I32, int64(i)))

		// Data pointer'ını al (index 0)
		dataIndices := []value.Value{
			constant.NewInt(types.I32, 0),
			constant.NewInt(types.I32, 0),
		}
		dataPtr := g.currentBB.NewGetElementPtr(ptrType.ElemType, sliceArg, dataIndices...)
		dataArray := g.currentBB.NewLoad(types.NewPointer(elementType), dataPtr)

		// Element adresini hesapla
		elementPtr := g.currentBB.NewGetElementPtr(elementType, dataArray, elementIndex)

		// Element'i store et
		g.currentBB.NewStore(elementVal, elementPtr)
	}

	// Length'i güncelle
	g.currentBB.NewStore(newLen, lenPtr)
	g.currentBB.NewBr(endBlock)

	// End block
	g.currentBB = endBlock

	// Güncellenmiş slice'ı döndür
	return sliceArg
}

// generateMakeCall, make() built-in function için IR üretir.
func (g *IRGenerator) generateMakeCall(args []ast.Expression) value.Value {
	if len(args) < 1 || len(args) > 3 {
		g.ReportError("make() fonksiyonu 1-3 argüman alır, %d verildi", len(args))
		return nil
	}

	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, make() çağrısı yapılamıyor")
		return nil
	}

	// İlk argüman tip olmalı
	typeExpr := args[0]

	// Tip ifadesini analiz et
	switch t := typeExpr.(type) {
	case *ast.ArrayType:
		// Slice oluştur: make([]T, len, cap)
		if t.Size != nil {
			g.ReportError("make() fonksiyonu array tipi ile kullanılamaz, slice tipi kullanın")
			return nil
		}

		// Element tipini belirle
		var elementType types.Type = types.I32 // Varsayılan
		if elemIdent, ok := t.ElementType.(*ast.Identifier); ok {
			if et, exists := g.typeTable[elemIdent.Value]; exists {
				elementType = et
			}
		}

		// Length argümanı (zorunlu)
		var length value.Value
		if len(args) >= 2 {
			length = g.generateExpression(args[1])
		} else {
			g.ReportError("make() slice için length argümanı gerekli")
			return nil
		}

		// Capacity argümanı (opsiyonel, varsayılan length ile aynı)
		var capacity value.Value
		if len(args) >= 3 {
			capacity = g.generateExpression(args[2])
		} else {
			capacity = length
		}

		// Slice struct oluştur: {data *T, len int32, cap int32}
		sliceStructType := types.NewStruct(
			types.NewPointer(elementType), // data
			types.I32,                     // len
			types.I32,                     // cap
		)

		// Slice için bellek ayır
		sliceAlloca := g.currentBB.NewAlloca(sliceStructType)

		// Data array için bellek ayır (basit implementasyon)
		// TODO: Gerçek heap allocation implementasyonu
		dataArrayType := types.NewArray(1024, elementType) // Sabit boyut
		dataAlloca := g.currentBB.NewAlloca(dataArrayType)
		dataPtr := g.currentBB.NewGetElementPtr(dataArrayType, dataAlloca,
			constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))

		// Slice struct'ını doldur
		// data field (index 0)
		dataFieldPtr := g.currentBB.NewGetElementPtr(sliceStructType, sliceAlloca,
			constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 0))
		g.currentBB.NewStore(dataPtr, dataFieldPtr)

		// len field (index 1)
		lenFieldPtr := g.currentBB.NewGetElementPtr(sliceStructType, sliceAlloca,
			constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 1))
		g.currentBB.NewStore(length, lenFieldPtr)

		// cap field (index 2)
		capFieldPtr := g.currentBB.NewGetElementPtr(sliceStructType, sliceAlloca,
			constant.NewInt(types.I32, 0), constant.NewInt(types.I32, 2))
		g.currentBB.NewStore(capacity, capFieldPtr)

		return sliceAlloca

	default:
		g.ReportError("make() fonksiyonu desteklenmeyen tip ile kullanıldı: %T", typeExpr)
		return nil
	}
}

// String operations helper functions

// isStringType, bir tipin string tipi olup olmadığını kontrol eder.
func (g *IRGenerator) isStringType(t types.Type) bool {
	// String'i pointer to i8 array olarak handle ediyoruz
	if ptrType, ok := t.(*types.PointerType); ok {
		if arrType, ok := ptrType.ElemType.(*types.ArrayType); ok {
			return arrType.ElemType == types.I8
		}
		// Ayrıca direkt i8 pointer'ı da string olarak kabul et
		return ptrType.ElemType == types.I8
	}
	return false
}

// generateStringConcatenation, iki string'i birleştirmek için IR üretir.
func (g *IRGenerator) generateStringConcatenation(left, right value.Value) value.Value {
	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, string concatenation yapılamıyor")
		return nil
	}

	// strcat fonksiyonunu bul veya tanımla
	strcatFunc := g.getFunction("strcat")
	if strcatFunc == nil {
		// strcat(char *dest, const char *src) -> char*
		strcatFunc = g.module.NewFunc("strcat",
			types.NewPointer(types.I8),
			ir.NewParam("dest", types.NewPointer(types.I8)),
			ir.NewParam("src", types.NewPointer(types.I8)))
		g.symbolTable["strcat"] = strcatFunc
	}

	// strlen fonksiyonunu bul veya tanımla (buffer size hesaplamak için)
	strlenFunc := g.getFunction("strlen")
	if strlenFunc == nil {
		// strlen(const char *str) -> size_t (i32 olarak handle edelim)
		strlenFunc = g.module.NewFunc("strlen",
			types.I32,
			ir.NewParam("str", types.NewPointer(types.I8)))
		g.symbolTable["strlen"] = strlenFunc
	}

	// malloc fonksiyonunu bul veya tanımla
	mallocFunc := g.getFunction("malloc")
	if mallocFunc == nil {
		// malloc(size_t size) -> void*
		mallocFunc = g.module.NewFunc("malloc",
			types.NewPointer(types.I8),
			ir.NewParam("size", types.I32))
		g.symbolTable["malloc"] = mallocFunc
	}

	// strcpy fonksiyonunu bul veya tanımla
	strcpyFunc := g.getFunction("strcpy")
	if strcpyFunc == nil {
		// strcpy(char *dest, const char *src) -> char*
		strcpyFunc = g.module.NewFunc("strcpy",
			types.NewPointer(types.I8),
			ir.NewParam("dest", types.NewPointer(types.I8)),
			ir.NewParam("src", types.NewPointer(types.I8)))
		g.symbolTable["strcpy"] = strcpyFunc
	}

	// String uzunluklarını hesapla
	leftLen := g.currentBB.NewCall(strlenFunc, left)
	rightLen := g.currentBB.NewCall(strlenFunc, right)

	// Toplam uzunluk + null terminator için 1
	totalLen := g.currentBB.NewAdd(leftLen, rightLen)
	totalLen = g.currentBB.NewAdd(totalLen, constant.NewInt(types.I32, 1))

	// Yeni buffer ayır
	newBuffer := g.currentBB.NewCall(mallocFunc, totalLen)

	// İlk string'i kopyala
	g.currentBB.NewCall(strcpyFunc, newBuffer, left)

	// İkinci string'i ekle
	result := g.currentBB.NewCall(strcatFunc, newBuffer, right)

	return result
}

// generateBoundsCheck, array/slice indexing için bounds checking IR'ı üretir.
func (g *IRGenerator) generateBoundsCheck(index, length value.Value) {
	if g.currentBB == nil {
		g.ReportError("Geçerli bir blok yok, bounds check yapılamıyor")
		return
	}

	// Index < 0 kontrolü
	negativeCheck := g.currentBB.NewICmp(enum.IPredSLT, index, constant.NewInt(types.I32, 0))

	// Index >= length kontrolü
	boundsCheck := g.currentBB.NewICmp(enum.IPredSGE, index, length)

	// Herhangi bir koşul true ise panic
	outOfBounds := g.currentBB.NewOr(negativeCheck, boundsCheck)

	// Panic ve normal execution blokları oluştur
	panicBlock := g.currentFunc.NewBlock("bounds_panic")
	normalBlock := g.currentFunc.NewBlock("bounds_ok")

	// Koşullu dallanma
	g.currentBB.NewCondBr(outOfBounds, panicBlock, normalBlock)

	// Panic block - runtime error
	g.currentBB = panicBlock

	// panic fonksiyonunu bul veya tanımla
	panicFunc := g.getFunction("panic")
	if panicFunc == nil {
		// panic(const char* message) -> void
		panicFunc = g.module.NewFunc("panic",
			types.Void,
			ir.NewParam("message", types.NewPointer(types.I8)))
		g.symbolTable["panic"] = panicFunc
	}

	// Error message oluştur
	errorMsg := g.generateStringLiteral(&ast.StringLiteral{Value: "runtime error: index out of range"})

	// panic çağrısı
	g.currentBB.NewCall(panicFunc, errorMsg)

	// panic'ten sonra unreachable
	g.currentBB.NewUnreachable()

	// Normal execution'a devam et
	g.currentBB = normalBlock
}
